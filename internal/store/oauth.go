package store

type OAuthAccount struct {
	Provider   string `json:"provider"`
	ProviderID string `json:"provider_id"`
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	AvatarURL  string `json:"avatar_url"`
}

type OAuthStore interface {
	GetOAuthAccount(provider, providerID string) (*OAuthAccount, error)
	CreateOAuthAccount(a *OAuthAccount) error
	DeleteOAuthAccount(provider, providerID string) error
	ListOAuthAccountsByUser(userID string) ([]OAuthAccount, error)
}
