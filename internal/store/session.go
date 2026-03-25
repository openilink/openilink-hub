package store

import "time"

type SessionStore interface {
	CreateSession(token, userID string, expiresAt time.Time) error
	GetSession(token string) (userID string, expiresAt time.Time, err error)
	DeleteSession(token string) error
	DeleteExpiredSessions() error
	DeleteSessionsByUserID(userID string) error
}
