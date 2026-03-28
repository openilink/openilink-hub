package store

type Credential struct {
	ID              string
	UserID          string
	Name            string
	PublicKey       []byte
	AttestationType string
	Transport       string
	SignCount       uint32
	BackupEligible  bool
	BackupState     bool
	CreatedAt       int64
}

type CredentialStore interface {
	SaveCredential(c *Credential) error
	GetCredentialsByUserID(userID string) ([]Credential, error)
	UpdateCredentialSignCount(id string, signCount uint32) error
	UpdateCredentialName(id, userID, name string) (bool, error)
	DeleteCredential(id, userID string) error
}
