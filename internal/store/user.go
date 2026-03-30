package store

import (
	"fmt"
	"regexp"
)

const (
	RoleSuperAdmin = "superadmin"
	RoleAdmin      = "admin"
	RoleMember     = "member"

	StatusActive   = "active"
	StatusDisabled = "disabled"
)

// IsAdmin returns true if the user has admin or superadmin role.
func IsAdmin(role string) bool {
	return role == RoleSuperAdmin || role == RoleAdmin
}

var usernameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*[a-z0-9]$`)

var reservedUsernames = map[string]bool{
	"admin": true, "administrator": true, "superadmin": true,
	"root": true, "system": true, "api": true, "support": true,
}

// ValidateUsername checks that a username meets format requirements.
func ValidateUsername(username string) error {
	n := len(username)
	if n < 2 || n > 32 {
		return fmt.Errorf("用户名长度需要 2-32 个字符")
	}
	if n == 1 {
		if !regexp.MustCompile(`^[a-z0-9]$`).MatchString(username) {
			return fmt.Errorf("用户名只能包含小写字母、数字、下划线和连字符")
		}
	} else if !usernameRegexp.MatchString(username) {
		return fmt.Errorf("用户名只能包含小写字母、数字、下划线和连字符，且不能以 _ 或 - 开头结尾")
	}
	if reservedUsernames[username] {
		return fmt.Errorf("该用户名为系统保留名称")
	}
	return nil
}

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email,omitempty"`
	DisplayName  string `json:"display_name"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

type UserStore interface {
	CreateUser(username, displayName string) (*User, error)
	CreateUserFull(username, email, displayName, passwordHash, role string) (*User, error)
	GetUserByID(id string) (*User, error)
	GetUserByUsername(username string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	ListUsers() ([]User, error)
	UserCount() (int, error)
	UpdateUserProfile(id, displayName, email string) error
	UpdateUserPassword(id, passwordHash string) error
	UpdateUserRole(id, role string) error
	UpdateUserStatus(id, status string) error
	UpdateUserUsername(id, username string) error
	DeleteUser(id string) error
}
