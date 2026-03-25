package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

const sessionTTL = 7 * 24 * time.Hour

func generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func CreateSession(s store.SessionStore, userID string) (string, error) {
	token := generateToken()
	expiresAt := time.Now().Add(sessionTTL)
	err := s.CreateSession(token, userID, expiresAt)
	if err != nil {
		return "", err
	}
	return token, nil
}

func ValidateSession(s store.SessionStore, token string) (string, error) {
	userID, expiresAt, err := s.GetSession(token)
	if err != nil {
		return "", err
	}
	if time.Now().After(expiresAt) {
		s.DeleteSession(token)
		return "", err
	}
	return userID, nil
}

func DeleteSession(s store.SessionStore, token string) {
	s.DeleteSession(token)
}

func CleanExpiredSessions(s store.SessionStore) {
	s.DeleteExpiredSessions()
}
