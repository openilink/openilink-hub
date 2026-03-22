package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/openilink/openilink-hub/internal/database"
)

const sessionTTL = 7 * 24 * time.Hour

func generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func CreateSession(db *database.DB, userID string) (string, error) {
	token := generateToken()
	expiresAt := time.Now().Add(sessionTTL).Unix()
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)", token, userID, expiresAt)
	if err != nil {
		return "", err
	}
	return token, nil
}

func ValidateSession(db *database.DB, token string) (string, error) {
	var userID string
	var expiresAt int64
	err := db.QueryRow("SELECT user_id, expires_at FROM sessions WHERE token = ?", token).Scan(&userID, &expiresAt)
	if err != nil {
		return "", err
	}
	if time.Now().Unix() > expiresAt {
		db.Exec("DELETE FROM sessions WHERE token = ?", token)
		return "", err
	}
	return userID, nil
}

func DeleteSession(db *database.DB, token string) {
	db.Exec("DELETE FROM sessions WHERE token = ?", token)
}

func CleanExpiredSessions(db *database.DB) {
	db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now().Unix())
}
