package auth
import (
	"time"
)
type User struct {
	ID       string                 `json:"id"`
	Email    string                 `json:"email"`
	Password []byte                 `json:"-"`
	Verified bool                   `json:"verified"`
	Traits   map[string]interface{} `json:"traits,omitempty"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}
type UserStore interface {
	AddUser(email, password string, traits map[string]interface{}) (*User, error)
	GetUserByEmail(email string) (*User, bool)
	GetUserByID(id string) (*User, bool)
	UpdatePassword(email, newPassword string) error
	VerifyUserEmail(email string) error
	UpdateUserTraits(id string, traits map[string]interface{}) error
	DeleteUser(id string) error
	ListUsers(offset, limit int) ([]*User, error)
}
type TokenStore interface {
	AddToken(jti string) error
	UseToken(jti string) bool
	IsTokenUsed(jti string) bool
	CleanupExpiredTokens() error
}
const (
	UserIDKey = "user_id"
)