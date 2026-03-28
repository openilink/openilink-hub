package sqlite

import "time"

func (db *DB) CreateSession(token, userID string, expiresAt time.Time) error {
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)", token, userID, expiresAt.Unix())
	return err
}

func (db *DB) GetSession(token string) (string, time.Time, error) {
	var userID string
	var epoch int64
	err := db.QueryRow("SELECT user_id, expires_at FROM sessions WHERE token = ?", token).Scan(&userID, &epoch)
	return userID, time.Unix(epoch, 0), err
}

func (db *DB) DeleteSession(token string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func (db *DB) DeleteExpiredSessions() error {
	_, err := db.Exec("DELETE FROM sessions WHERE expires_at < ?", db.now())
	return err
}

func (db *DB) DeleteSessionsByUserID(userID string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}
