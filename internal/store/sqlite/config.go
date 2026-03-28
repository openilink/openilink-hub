package sqlite

import "database/sql"

func (db *DB) GetConfig(key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM system_config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (db *DB) SetConfig(key, value string) error {
	now := db.now()
	_, err := db.Exec(`
		INSERT INTO system_config (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (key) DO UPDATE SET value = ?, updated_at = ?`,
		key, value, now, value, now,
	)
	return err
}

func (db *DB) DeleteConfig(key string) error {
	_, err := db.Exec("DELETE FROM system_config WHERE key = ?", key)
	return err
}

func (db *DB) ListConfigByPrefix(prefix string) (map[string]string, error) {
	rows, err := db.Query("SELECT key, value FROM system_config WHERE key LIKE ?", prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}
