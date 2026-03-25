package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/openilink/openilink-hub/internal/api"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/bot"
	"github.com/openilink/openilink-hub/internal/config"
	"github.com/openilink/openilink-hub/internal/relay"
	"github.com/openilink/openilink-hub/internal/sink"
	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/store/postgres"
	"github.com/openilink/openilink-hub/internal/store/sqlite"
	"github.com/openilink/openilink-hub/internal/storage"

	// Register providers
	_ "github.com/openilink/openilink-hub/internal/provider/ilink"
)

// Set by goreleaser ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func openStore(dsn string) (store.Store, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return postgres.Open(dsn)
	}
	return sqlite.Open(dsn)
}

func main() {
	cfg := config.Parse()

	// Database
	s, err := openStore(cfg.DBPath)
	if err != nil {
		slog.Error("database open failed", "err", err)
		os.Exit(1)
	}
	defer s.Close()

	// WebAuthn
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: cfg.RPName,
		RPID:          cfg.RPID,
		RPOrigins:     []string{cfg.RPOrigin},
	})
	if err != nil {
		slog.Error("webauthn init failed", "err", err)
		os.Exit(1)
	}

	// Server components
	srv := &api.Server{
		Store:        s,
		WebAuthn:     wa,
		SessionStore: auth.NewSessionStore(),
		Config:       cfg,
		OAuthStates:  api.SetupOAuth(cfg),
	}

	// Storage (optional)
	var objStore *storage.Storage
	if cfg.StorageEndpoint != "" {
		var err error
		publicURL := cfg.StoragePublicURL
		if publicURL == "" {
			publicURL = cfg.RPOrigin + "/api/v1/media"
		}
		objStore, err = storage.New(storage.Config{
			Endpoint:  cfg.StorageEndpoint,
			AccessKey: cfg.StorageAccessKey,
			SecretKey: cfg.StorageSecretKey,
			Bucket:    cfg.StorageBucket,
			UseSSL:    cfg.StorageSSL,
			PublicURL: publicURL,
		})
		if err != nil {
			slog.Error("storage init failed", "err", err)
			os.Exit(1)
		}
		slog.Info("storage connected", "endpoint", cfg.StorageEndpoint, "bucket", cfg.StorageBucket)
		srv.ObjectStore = objStore
	}

	hub := relay.NewHub(srv.SetupUpstreamHandler())
	sinks := []sink.Sink{
		&sink.WS{Hub: hub},
		&sink.AI{Store: s},
		&sink.Webhook{Store: s},
	}
	mgr := bot.NewManager(s, hub, sinks, objStore, cfg.RPOrigin)
	srv.BotManager = mgr
	srv.Hub = hub

	// Start all saved bots
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	mgr.StartAll(ctx)

	// Periodic cleanup
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				auth.CleanExpiredSessions(s)
			}
		}
	}()

	// HTTP server
	httpSrv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: srv.Handler(),
	}

	go func() {
		<-ctx.Done()
		slog.Info("shutting down...")
		mgr.StopAll()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		httpSrv.Shutdown(shutCtx)
	}()

	fmt.Printf("OpeniLink Hub %s (%s, %s) running on http://localhost%s\n", version, commit, date, cfg.ListenAddr)
	if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
