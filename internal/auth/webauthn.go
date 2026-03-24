package auth

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/openilink/openilink-hub/internal/database"
)

// WebAuthnUser adapts our DB user to the webauthn.User interface.
type WebAuthnUser struct {
	user  *database.User
	creds []webauthn.Credential
}

func (u *WebAuthnUser) WebAuthnID() []byte                         { return []byte(u.user.ID) }
func (u *WebAuthnUser) WebAuthnName() string                       { return u.user.Username }
func (u *WebAuthnUser) WebAuthnDisplayName() string                { return u.user.DisplayName }
func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }

// SessionStore keeps in-flight WebAuthn ceremony data (short-lived).
type SessionStore struct {
	mu    sync.Mutex
	store map[string]*sessionEntry
}

type sessionEntry struct {
	data      *webauthn.SessionData
	createdAt time.Time
}

func NewSessionStore() *SessionStore {
	return &SessionStore{store: make(map[string]*sessionEntry)}
}

func (s *SessionStore) Set(key string, data *webauthn.SessionData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[key] = &sessionEntry{data: data, createdAt: time.Now()}
}

func (s *SessionStore) Get(key string) *webauthn.SessionData {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.store[key]
	if !ok || time.Since(e.createdAt) > 5*time.Minute {
		delete(s.store, key)
		return nil
	}
	delete(s.store, key)
	return e.data
}

// LoadWebAuthnUser loads a user and their credentials from the DB.
func LoadWebAuthnUser(db *database.DB, user *database.User) (*WebAuthnUser, error) {
	dbCreds, err := db.GetCredentialsByUserID(user.ID)
	if err != nil {
		return nil, err
	}

	var creds []webauthn.Credential
	for _, dc := range dbCreds {
		var transports []protocol.AuthenticatorTransport
		_ = json.Unmarshal([]byte(dc.Transport), &transports)

		creds = append(creds, webauthn.Credential{
			ID:              []byte(dc.ID),
			PublicKey:       dc.PublicKey,
			AttestationType: dc.AttestationType,
			Transport:       transports,
			Authenticator: webauthn.Authenticator{
				SignCount: dc.SignCount,
			},
		})
	}

	return &WebAuthnUser{user: user, creds: creds}, nil
}
