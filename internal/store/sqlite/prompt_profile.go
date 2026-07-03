package sqlite

import (
	"database/sql"

	"github.com/openilink/openilink-hub/internal/store"
)

const promptProfileSelectCols = `id, bot_id, sender_user_id, binding_id,
	system_prompt, user_prompt, full_prompt, full_prompt_hash,
	prompt_version, source_updated_at, status, created_at, updated_at`

func scanPromptProfile(scanner interface{ Scan(...any) error }) (*store.PromptProfile, error) {
	p := &store.PromptProfile{}
	if err := scanner.Scan(
		&p.ID,
		&p.BotID,
		&p.SenderUserID,
		&p.BindingID,
		&p.SystemPrompt,
		&p.UserPrompt,
		&p.FullPrompt,
		&p.FullPromptHash,
		&p.PromptVersion,
		&p.SourceUpdatedAt,
		&p.Status,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return p, nil
}

func (db *DB) UpsertPromptProfile(in store.PromptProfileUpsertInput) (*store.PromptProfile, bool, error) {
	status := store.NormalizePromptProfileStatus(in.Status)
	hash := ""
	if !store.IsBlankPrompt(in.FullPrompt) {
		hash = store.HashPrompt(in.FullPrompt)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()

	curr, _ := scanPromptProfile(tx.QueryRow(
		"SELECT "+promptProfileSelectCols+" FROM prompt_profiles WHERE bot_id = ? AND sender_user_id = ? AND status = 'active' LIMIT 1",
		in.BotID, in.SenderUserID,
	))
	if curr != nil && !store.IsValidPromptVersion(in.PromptVersion, in.SourceUpdatedAt, curr.PromptVersion, curr.SourceUpdatedAt) {
		if err := tx.Commit(); err != nil {
			return nil, false, err
		}
		return curr, false, nil
	}

	if status == store.PromptProfileStatusActive {
		if _, err := tx.Exec(
			"UPDATE prompt_profiles SET status = 'inactive', updated_at = ? WHERE bot_id = ? AND sender_user_id = ? AND status = 'active' AND binding_id != ?",
			db.now(), in.BotID, in.SenderUserID, in.BindingID,
		); err != nil {
			return nil, false, err
		}
	}

	res, err := tx.Exec(`INSERT INTO prompt_profiles (
		bot_id, sender_user_id, binding_id,
		system_prompt, user_prompt, full_prompt, full_prompt_hash,
		prompt_version, source_updated_at, status, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(bot_id, sender_user_id, binding_id) DO UPDATE SET
		system_prompt = excluded.system_prompt,
		user_prompt = excluded.user_prompt,
		full_prompt = excluded.full_prompt,
		full_prompt_hash = excluded.full_prompt_hash,
		prompt_version = excluded.prompt_version,
		source_updated_at = excluded.source_updated_at,
		status = excluded.status,
		updated_at = excluded.updated_at
	WHERE excluded.prompt_version > prompt_profiles.prompt_version
		OR (excluded.prompt_version = prompt_profiles.prompt_version
			AND excluded.source_updated_at > prompt_profiles.source_updated_at)`,
		in.BotID, in.SenderUserID, in.BindingID,
		in.SystemPrompt, in.UserPrompt, in.FullPrompt, hash,
		in.PromptVersion, in.SourceUpdatedAt, status, db.now(),
	)
	if err != nil {
		return nil, false, err
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		p, err := scanPromptProfile(tx.QueryRow(
			"SELECT "+promptProfileSelectCols+" FROM prompt_profiles WHERE bot_id = ? AND sender_user_id = ? AND binding_id = ?",
			in.BotID, in.SenderUserID, in.BindingID,
		))
		if err != nil {
			return nil, false, err
		}
		if err := tx.Commit(); err != nil {
			return nil, false, err
		}
		return p, false, nil
	}

	p, err := scanPromptProfile(tx.QueryRow(
		"SELECT "+promptProfileSelectCols+" FROM prompt_profiles WHERE bot_id = ? AND sender_user_id = ? AND binding_id = ?",
		in.BotID, in.SenderUserID, in.BindingID,
	))
	if err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return p, true, nil
}

func (db *DB) GetActivePromptProfile(botID, senderUserID string) (*store.PromptProfile, error) {
	p, err := scanPromptProfile(db.QueryRow(
		"SELECT "+promptProfileSelectCols+" FROM prompt_profiles WHERE bot_id = ? AND sender_user_id = ? AND status = 'active' LIMIT 1",
		botID, senderUserID,
	))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (db *DB) InvalidatePromptProfile(botID, senderUserID, bindingID string) (bool, error) {
	q := "UPDATE prompt_profiles SET status = 'inactive', updated_at = ? WHERE bot_id = ? AND sender_user_id = ? AND status = 'active'"
	args := []any{db.now(), botID, senderUserID}
	if bindingID != "" {
		q += " AND binding_id = ?"
		args = append(args, bindingID)
	}
	res, err := db.Exec(q, args...)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (db *DB) GetPromptProfile(botID, senderUserID, bindingID string) (*store.PromptProfile, error) {
	p, err := scanPromptProfile(db.QueryRow(
		"SELECT "+promptProfileSelectCols+" FROM prompt_profiles WHERE bot_id = ? AND sender_user_id = ? AND binding_id = ?",
		botID, senderUserID, bindingID,
	))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}
