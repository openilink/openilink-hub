package postgres

import "github.com/openilink/openilink-hub/internal/store"

func (db *DB) SaveCredential(c *store.Credential) error {
	_, err := db.Exec(
		`INSERT INTO credentials (id, user_id, name, public_key, attestation_type, transport, sign_count, backup_eligible, backup_state)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (id) DO UPDATE SET user_id = $2, name = $3, public_key = $4, attestation_type = $5, transport = $6, sign_count = $7, backup_eligible = $8, backup_state = $9`,
		c.ID, c.UserID, c.Name, c.PublicKey, c.AttestationType, c.Transport, c.SignCount, c.BackupEligible, c.BackupState,
	)
	return err
}

func (db *DB) GetCredentialsByUserID(userID string) ([]store.Credential, error) {
	rows, err := db.Query(
		`SELECT id, user_id, name, public_key, attestation_type, transport, sign_count, backup_eligible, backup_state, EXTRACT(EPOCH FROM created_at)::BIGINT
		 FROM credentials WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var creds []store.Credential
	for rows.Next() {
		var c store.Credential
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.PublicKey, &c.AttestationType, &c.Transport, &c.SignCount, &c.BackupEligible, &c.BackupState, &c.CreatedAt); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

func (db *DB) UpdateCredentialSignCount(id string, signCount uint32) error {
	_, err := db.Exec("UPDATE credentials SET sign_count = $1 WHERE id = $2", signCount, id)
	return err
}

func (db *DB) UpdateCredentialName(id, userID, name string) error {
	_, err := db.Exec("UPDATE credentials SET name = $1 WHERE id = $2 AND user_id = $3", name, id, userID)
	return err
}

func (db *DB) DeleteCredential(id, userID string) error {
	_, err := db.Exec("DELETE FROM credentials WHERE id = $1 AND user_id = $2", id, userID)
	return err
}
