package postgres

import "time"

func (db *DB) CreateSession(token, userID string, expiresAt time.Time) error {
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)", token, userID, expiresAt)
	return err
}

func (db *DB) GetSession(token string) (string, time.Time, error) {
	var userID string
	var expiresAt time.Time
	err := db.QueryRow("SELECT user_id, expires_at FROM sessions WHERE token = $1", token).Scan(&userID, &expiresAt)
	return userID, expiresAt, err
}

func (db *DB) DeleteSession(token string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE token = $1", token)
	return err
}

func (db *DB) DeleteExpiredSessions() error {
	_, err := db.Exec("DELETE FROM sessions WHERE expires_at < NOW()")
	return err
}

func (db *DB) DeleteSessionsByUserID(userID string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE user_id = $1", userID)
	return err
}
