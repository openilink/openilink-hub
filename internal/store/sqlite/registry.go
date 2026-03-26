package sqlite

import (
	"github.com/google/uuid"
	"github.com/openilink/openilink-hub/internal/store"
)

func (db *DB) ListRegistries() ([]store.Registry, error) {
	rows, err := db.Query(`SELECT id, name, url, enabled, created_at FROM registries ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []store.Registry
	for rows.Next() {
		var r store.Registry
		if err := rows.Scan(&r.ID, &r.Name, &r.URL, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (db *DB) CreateRegistry(r *store.Registry) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	_, err := db.Exec(`INSERT INTO registries (id, name, url) VALUES (?, ?, ?)`,
		r.ID, r.Name, r.URL)
	if err != nil {
		return err
	}
	return db.QueryRow("SELECT created_at FROM registries WHERE id = ?", r.ID).Scan(&r.CreatedAt)
}

func (db *DB) UpdateRegistryEnabled(id string, enabled bool) error {
	_, err := db.Exec("UPDATE registries SET enabled = ? WHERE id = ?", enabled, id)
	return err
}

func (db *DB) DeleteRegistry(id string) error {
	_, err := db.Exec("DELETE FROM registries WHERE id = ?", id)
	return err
}
