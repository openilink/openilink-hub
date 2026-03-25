package sink

import (
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/relay"
	"github.com/openilink/openilink-hub/internal/store"
)

// Delivery holds all context for delivering a message to a channel sink.
type Delivery struct {
	BotDBID      string
	Provider     provider.Provider
	Channel      store.Channel
	Message      provider.InboundMessage
	Envelope     relay.Envelope
	SeqID        int64
	MsgType      string
	Content      string
}

// Sink processes messages delivered to a channel.
type Sink interface {
	Name() string
	Handle(d Delivery)
}
