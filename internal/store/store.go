package store

import "io"

// Store is the aggregate interface for all storage operations.
type Store interface {
	UserStore
	BotStore
	MessageStore
	ChannelStore
	AppStore
	PluginStore
	TraceStore
	CredentialStore
	OAuthStore
	ConfigStore
	WebhookLogStore
	AppLogStore
	SessionStore
	io.Closer
}
