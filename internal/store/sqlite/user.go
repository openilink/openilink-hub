package sqlite

import (
	"github.com/google/uuid"
	"github.com/openilink/openilink-hub/internal/store"
)

const userSelectCols = `id, username, email, display_name, password_hash, role, status,
	created_at, updated_at`

func scanUser(scanner interface{ Scan(...any) error }) (*store.User, error) {
	u := &store.User{}
	err := scanner.Scan(&u.ID, &u.Username, &u.Email, &u.DisplayName,
		&u.PasswordHash, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) CreateUser(username, displayName string) (*store.User, error) {
	id := uuid.New().String()
	role := store.RoleMember
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count == 0 {
		role = store.RoleSuperAdmin
	}
	_, err := db.Exec(
		"INSERT INTO users (id, username, display_name, role) VALUES (?, ?, ?, ?)",
		id, username, displayName, role,
	)
	if err != nil {
		return nil, err
	}
	return &store.User{ID: id, Username: username, DisplayName: displayName, Role: role, Status: store.StatusActive}, nil
}

func (db *DB) CreateUserFull(username, email, displayName, passwordHash, role string) (*store.User, error) {
	id := uuid.New().String()
	if role == "" {
		role = store.RoleMember
	}
	_, err := db.Exec(
		`INSERT INTO users (id, username, email, display_name, password_hash, role)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, username, email, displayName, passwordHash, role,
	)
	if err != nil {
		return nil, err
	}
	return &store.User{ID: id, Username: username, Email: email, DisplayName: displayName, Role: role, Status: store.StatusActive}, nil
}

func (db *DB) GetUserByID(id string) (*store.User, error) {
	return scanUser(db.QueryRow("SELECT "+userSelectCols+" FROM users WHERE id = ?", id))
}

func (db *DB) GetUserByUsername(username string) (*store.User, error) {
	return scanUser(db.QueryRow("SELECT "+userSelectCols+" FROM users WHERE username = ?", username))
}

func (db *DB) GetUserByEmail(email string) (*store.User, error) {
	return scanUser(db.QueryRow("SELECT "+userSelectCols+" FROM users WHERE email = ?", email))
}

func (db *DB) ListUsers() ([]store.User, error) {
	rows, err := db.Query("SELECT " + userSelectCols + " FROM users ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []store.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (db *DB) UserCount() (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (db *DB) UpdateUserProfile(id, displayName, email string) error {
	_, err := db.Exec(
		"UPDATE users SET display_name = ?, email = ?, updated_at = ? WHERE id = ?",
		displayName, email, db.now(), id,
	)
	return err
}

func (db *DB) UpdateUserPassword(id, passwordHash string) error {
	_, err := db.Exec(
		"UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?",
		passwordHash, db.now(), id,
	)
	return err
}

func (db *DB) UpdateUserRole(id, role string) error {
	_, err := db.Exec(
		"UPDATE users SET role = ?, updated_at = ? WHERE id = ?",
		role, db.now(), id,
	)
	return err
}

func (db *DB) UpdateUserStatus(id, status string) error {
	_, err := db.Exec(
		"UPDATE users SET status = ?, updated_at = ? WHERE id = ?",
		status, db.now(), id,
	)
	return err
}

func (db *DB) DeleteUser(id string) error {
	db.Exec("DELETE FROM oauth_accounts WHERE user_id = ?", id)
	db.Exec("DELETE FROM sessions WHERE user_id = ?", id)
	db.Exec("DELETE FROM credentials WHERE user_id = ?", id)
	_, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}
