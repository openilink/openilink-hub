package sqlite

import "github.com/openilink/openilink-hub/internal/store"

func (db *DB) SaveCredential(c *store.Credential) error {
	_, err := db.Exec(
		`INSERT INTO credentials (id, user_id, public_key, attestation_type, transport, sign_count, backup_eligible, backup_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET user_id = ?, public_key = ?, attestation_type = ?, transport = ?, sign_count = ?, backup_eligible = ?, backup_state = ?, created_at = ?`,
		c.ID, c.UserID, c.PublicKey, c.AttestationType, c.Transport, c.SignCount, c.BackupEligible, c.BackupState,
		c.UserID, c.PublicKey, c.AttestationType, c.Transport, c.SignCount, c.BackupEligible, c.BackupState, db.now(),
	)
	return err
}

func (db *DB) GetCredentialsByUserID(userID string) ([]store.Credential, error) {
	rows, err := db.Query(
		"SELECT id, user_id, public_key, attestation_type, transport, sign_count, backup_eligible, backup_state, created_at FROM credentials WHERE user_id = ?",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var creds []store.Credential
	for rows.Next() {
		var c store.Credential
		if err := rows.Scan(&c.ID, &c.UserID, &c.PublicKey, &c.AttestationType, &c.Transport, &c.SignCount, &c.BackupEligible, &c.BackupState, &c.CreatedAt); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

func (db *DB) UpdateCredentialSignCount(id string, signCount uint32) error {
	_, err := db.Exec("UPDATE credentials SET sign_count = ? WHERE id = ?", signCount, id)
	return err
}

func (db *DB) DeleteCredential(id, userID string) error {
	_, err := db.Exec("DELETE FROM credentials WHERE id = ? AND user_id = ?", id, userID)
	return err
}
