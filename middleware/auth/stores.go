package auth

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"sync"
	"time"
)

type InMemoryUserStore struct {
	mu      sync.RWMutex
	users   map[string]*User
	byEmail map[string]*User
}

func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{
		users:   make(map[string]*User),
		byEmail: make(map[string]*User),
	}
}
func (s *InMemoryUserStore) AddUser(email, password string, traits map[string]interface{}) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byEmail[email]; exists {
		return nil, errors.New("user with this email already exists")
	}
	hashedPassword, err := SecureHashPassword(password, MinBcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}
	user := &User{
		ID:        uuid.New().String(),
		Email:     email,
		Password:  hashedPassword,
		Verified:  false,
		Traits:    traits,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.users[user.ID] = user
	s.byEmail[email] = user
	return user, nil
}
func (s *InMemoryUserStore) GetUserByEmail(email string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, exists := s.byEmail[email]
	if !exists {
		return nil, false
	}
	userCopy := *user
	return &userCopy, true
}
func (s *InMemoryUserStore) GetUserByID(id string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, exists := s.users[id]
	if !exists {
		return nil, false
	}
	userCopy := *user
	return &userCopy, true
}
func (s *InMemoryUserStore) UpdatePassword(email, newPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, exists := s.byEmail[email]
	if !exists {
		return errors.New("user not found")
	}
	hashedPassword, err := SecureHashPassword(newPassword, MinBcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}
	user.Password = hashedPassword
	user.UpdatedAt = time.Now()
	return nil
}
func (s *InMemoryUserStore) VerifyUserEmail(email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, exists := s.byEmail[email]
	if !exists {
		return errors.New("user not found")
	}
	user.Verified = true
	user.UpdatedAt = time.Now()
	return nil
}
func (s *InMemoryUserStore) UpdateUserTraits(id string, traits map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, exists := s.users[id]
	if !exists {
		return errors.New("user not found")
	}
	user.Traits = traits
	user.UpdatedAt = time.Now()
	return nil
}
func (s *InMemoryUserStore) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, exists := s.users[id]
	if !exists {
		return errors.New("user not found")
	}
	delete(s.users, id)
	delete(s.byEmail, user.Email)
	return nil
}
func (s *InMemoryUserStore) ListUsers(offset, limit int) ([]*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		userCopy := *user
		users = append(users, &userCopy)
	}
	start := offset
	if start >= len(users) {
		return []*User{}, nil
	}
	end := start + limit
	if end > len(users) {
		end = len(users)
	}
	return users[start:end], nil
}

type InMemoryTokenStore struct {
	mu      sync.RWMutex
	tokens  map[string]time.Time
	used    map[string]bool
	cleanup *time.Ticker
	stop    chan bool
}

func NewInMemoryTokenStore() *InMemoryTokenStore {
	store := &InMemoryTokenStore{
		tokens: make(map[string]time.Time),
		used:   make(map[string]bool),
		stop:   make(chan bool),
	}
	store.cleanup = time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-store.cleanup.C:
				store.cleanupExpiredTokens()
			case <-store.stop:
				store.cleanup.Stop()
				return
			}
		}
	}()
	return store
}
func (s *InMemoryTokenStore) AddToken(jti string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[jti] = time.Now()
	s.used[jti] = false
	return nil
}
func (s *InMemoryTokenStore) UseToken(jti string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tokens[jti]; !exists {
		return false
	}
	if s.used[jti] {
		return false
	}
	s.used[jti] = true
	return true
}
func (s *InMemoryTokenStore) IsTokenUsed(jti string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.used[jti]
}
func (s *InMemoryTokenStore) CleanupExpiredTokens() error {
	return s.cleanupExpiredTokens()
}
func (s *InMemoryTokenStore) cleanupExpiredTokens() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	expiry := 48 * time.Hour
	for jti, issuedAt := range s.tokens {
		if now.Sub(issuedAt) > expiry {
			delete(s.tokens, jti)
			delete(s.used, jti)
		}
	}
	return nil
}
func (s *InMemoryTokenStore) Stop() {
	close(s.stop)
}
func (s *InMemoryTokenStore) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	usedCount := 0
	for _, used := range s.used {
		if used {
			usedCount++
		}
	}
	return map[string]interface{}{
		"total_tokens":  len(s.tokens),
		"used_tokens":   usedCount,
		"active_tokens": len(s.tokens) - usedCount,
	}
}
