package auth

import "time"

// User represents an authenticated user
type User struct {
	ID        string                 `json:"id"`
	Email     string                 `json:"email"`
	Password  []byte                 `json:"-"`
	Verified  bool                   `json:"verified"`
	Traits    map[string]interface{} `json:"traits,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

const (
	UserIDKey = "user_id"
)

// UserStore defines the interface for user storage
type UserStore interface {
	GetUserByID(id string) (*User, bool)
	GetUserByEmail(email string) (*User, bool)
	AddUser(email, password string, traits map[string]interface{}) (*User, error)
	UpdatePassword(email, newPassword string) error
	VerifyUserEmail(email string) error
	UpdateUserTraits(id string, traits map[string]interface{}) error
	DeleteUser(id string) error
	ListUsers(offset, limit int) ([]*User, error)
}

// TokenStore defines the interface for token storage (for blacklist/revocation or stateful tokens)
type TokenStore interface {
	AddToken(jti string) error
	UseToken(jti string) bool
	IsTokenUsed(jti string) bool
	CleanupExpiredTokens() error
}
