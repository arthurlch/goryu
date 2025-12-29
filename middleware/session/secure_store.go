package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

type SecureStore struct {
	mu                   sync.RWMutex
	sessions             map[string]*encryptedSession
	encryptionKey        []byte
	gcm                  cipher.AEAD
	maxSize              int
	maxAge               time.Duration
	enableFingerprinting bool
	fingerprintFields    []string
	cleanupInterval      time.Duration
	stopCleanup          chan bool
}
type encryptedSession struct {
	Data        []byte
	Fingerprint string
	CreatedAt   time.Time
	AccessedAt  time.Time
	Version     int
}

func NewSecureStore(encryptionKey string, options ...SecureStoreOption) (*SecureStore, error) {
	if len(encryptionKey) < 32 {
		return nil, errors.New("encryption key must be at least 32 characters")
	}
	key := sha256.Sum256([]byte(encryptionKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	store := &SecureStore{
		sessions:          make(map[string]*encryptedSession),
		encryptionKey:     key[:],
		gcm:               gcm,
		maxSize:           1024 * 1024,
		maxAge:            24 * time.Hour,
		cleanupInterval:   1 * time.Hour,
		stopCleanup:       make(chan bool),
		fingerprintFields: []string{"User-Agent", "Accept-Language"},
	}
	for _, opt := range options {
		opt(store)
	}
	go store.cleanupRoutine()
	return store, nil
}

type SecureStoreOption func(*SecureStore)

func WithMaxSize(size int) SecureStoreOption {
	return func(s *SecureStore) {
		s.maxSize = size
	}
}
func WithMaxAge(age time.Duration) SecureStoreOption {
	return func(s *SecureStore) {
		s.maxAge = age
	}
}
func WithFingerprinting(fields ...string) SecureStoreOption {
	return func(s *SecureStore) {
		s.enableFingerprinting = true
		if len(fields) > 0 {
			s.fingerprintFields = fields
		}
	}
}
func (s *SecureStore) Get(id string) (*Session, error) {
	s.mu.RLock()
	encSession, exists := s.sessions[id]
	s.mu.RUnlock()
	if !exists {
		return nil, errors.New("session not found")
	}
	if time.Since(encSession.CreatedAt) > s.maxAge {
		s.mu.Lock()
		delete(s.sessions, id)
		s.mu.Unlock()
		return nil, errors.New("session expired")
	}
	decrypted, err := s.decrypt(encSession.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt session: %w", err)
	}
	var sessionData map[string]any
	if err := json.Unmarshal(decrypted, &sessionData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}
	s.mu.Lock()
	encSession.AccessedAt = time.Now()
	s.mu.Unlock()
	return &Session{
		ID:   id,
		Data: sessionData,
	}, nil
}
func (s *SecureStore) Save(session *Session) error {
	data, err := json.Marshal(session.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	if len(data) > s.maxSize {
		return errors.New("session data exceeds maximum size")
	}
	encrypted, err := s.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt session: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, exists := s.sessions[session.ID]; exists {
		s.sessions[session.ID] = &encryptedSession{
			Data:        encrypted,
			Fingerprint: existing.Fingerprint,
			CreatedAt:   existing.CreatedAt,
			AccessedAt:  time.Now(),
			Version:     existing.Version + 1,
		}
	} else {
		s.sessions[session.ID] = &encryptedSession{
			Data:       encrypted,
			CreatedAt:  time.Now(),
			AccessedAt: time.Now(),
			Version:    1,
		}
	}
	return nil
}
func (s *SecureStore) SaveWithFingerprint(session *Session, fingerprint string) error {
	err := s.Save(session)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if encSession, exists := s.sessions[session.ID]; exists {
		encSession.Fingerprint = fingerprint
	}
	s.mu.Unlock()
	return nil
}
func (s *SecureStore) ValidateFingerprint(id, fingerprint string) error {
	if !s.enableFingerprinting {
		return nil
	}
	s.mu.RLock()
	encSession, exists := s.sessions[id]
	s.mu.RUnlock()
	if !exists {
		return errors.New("session not found")
	}
	if encSession.Fingerprint != "" && encSession.Fingerprint != fingerprint {
		return errors.New("session fingerprint mismatch - possible hijacking attempt")
	}
	return nil
}
func (s *SecureStore) Destroy(id string) error {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
	return nil
}
func (s *SecureStore) encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := s.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}
func (s *SecureStore) decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < s.gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:s.gcm.NonceSize()], ciphertext[s.gcm.NonceSize():]
	plaintext, err := s.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
func (s *SecureStore) cleanupRoutine() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.stopCleanup:
			return
		}
	}
}
func (s *SecureStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, session := range s.sessions {
		if now.Sub(session.CreatedAt) > s.maxAge {
			delete(s.sessions, id)
		}
	}
}
func (s *SecureStore) Stop() {
	close(s.stopCleanup)
}
func GenerateFingerprint(headers map[string]string, fields []string) string {
	var parts []string
	for _, field := range fields {
		if value, exists := headers[field]; exists {
			parts = append(parts, field+"="+value)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	hash := sha256.Sum256([]byte(fmt.Sprintf("%v", parts)))
	return base64.StdEncoding.EncodeToString(hash[:])
}
