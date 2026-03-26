package postgres

import (
	"github.com/google/uuid"
	"github.com/openilink/openilink-hub/internal/store"
)

func (db *DB) ListRegistries() ([]store.Registry, error) {
	rows, err := db.Query(`SELECT id, name, url, enabled, EXTRACT(EPOCH FROM created_at)::BIGINT FROM registries ORDER BY name`)
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
	return db.QueryRow(`INSERT INTO registries (id, name, url) VALUES ($1, $2, $3)
		RETURNING EXTRACT(EPOCH FROM created_at)::BIGINT`,
		r.ID, r.Name, r.URL).Scan(&r.CreatedAt)
}

func (db *DB) UpdateRegistryEnabled(id string, enabled bool) error {
	_, err := db.Exec("UPDATE registries SET enabled = $1 WHERE id = $2", enabled, id)
	return err
}

func (db *DB) DeleteRegistry(id string) error {
	_, err := db.Exec("DELETE FROM registries WHERE id = $1", id)
	return err
}
