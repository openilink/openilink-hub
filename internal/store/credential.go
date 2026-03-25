package store

type Credential struct {
	ID              string
	UserID          string
	PublicKey       []byte
	AttestationType string
	Transport       string
	SignCount       uint32
	CreatedAt       int64
}

type CredentialStore interface {
	SaveCredential(c *Credential) error
	GetCredentialsByUserID(userID string) ([]Credential, error)
	UpdateCredentialSignCount(id string, signCount uint32) error
	DeleteCredential(id, userID string) error
}
