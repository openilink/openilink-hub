package store

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
	DeleteUser(id string) error
}
