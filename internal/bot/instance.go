package bot

import (
	"context"
	"log/slog"
	"sync/atomic"

	ilink "github.com/openilink/openilink-sdk-go"
	"github.com/openilink/openilink-hub/internal/database"
)

// Instance wraps an iLink client with its monitor goroutine.
type Instance struct {
	DBID   string // DB UUID
	BotID  string // ilink_bot_id
	Client *ilink.Client
	cancel context.CancelFunc
	status atomic.Value // string: "connected", "disconnected", "error"
}

func NewInstance(bot *database.Bot) *Instance {
	opts := []ilink.Option{}
	if bot.BaseURL != "" {
		opts = append(opts, ilink.WithBaseURL(bot.BaseURL))
	}
	client := ilink.NewClient(bot.BotToken, opts...)

	inst := &Instance{
		DBID:   bot.ID,
		BotID:  bot.BotID,
		Client: client,
	}
	inst.status.Store("disconnected")
	return inst
}

func (i *Instance) Status() string {
	return i.status.Load().(string)
}

func (i *Instance) SetStatus(s string) {
	i.status.Store(s)
}

// Start begins the Monitor loop in a goroutine. The onMessage callback
// is invoked for each inbound message.
func (i *Instance) Start(ctx context.Context, db *database.DB, onMessage func(inst *Instance, msg ilink.WeixinMessage)) {
	ctx, i.cancel = context.WithCancel(ctx)
	i.SetStatus("connected")
	_ = db.UpdateBotStatus(i.DBID, "connected")

	// Load saved sync buf
	bot, _ := db.GetBot(i.DBID)
	initialBuf := ""
	if bot != nil {
		initialBuf = bot.SyncBuf
	}

	go func() {
		err := i.Client.Monitor(ctx, func(msg ilink.WeixinMessage) {
			onMessage(i, msg)
		}, &ilink.MonitorOptions{
			InitialBuf: initialBuf,
			OnBufUpdate: func(buf string) {
				_ = db.UpdateBotSyncBuf(i.DBID, buf)
			},
			OnError: func(err error) {
				slog.Warn("bot monitor error", "bot", i.DBID, "err", err)
			},
			OnSessionExpired: func() {
				slog.Error("bot session expired", "bot", i.DBID)
				i.SetStatus("session_expired")
				_ = db.UpdateBotStatus(i.DBID, "session_expired")
			},
		})
		if err != nil && err != context.Canceled {
			slog.Error("bot monitor stopped", "bot", i.DBID, "err", err)
			i.SetStatus("error")
			_ = db.UpdateBotStatus(i.DBID, "error")
		} else {
			i.SetStatus("disconnected")
			_ = db.UpdateBotStatus(i.DBID, "disconnected")
		}
	}()
}

func (i *Instance) Stop() {
	if i.cancel != nil {
		i.cancel()
	}
}
