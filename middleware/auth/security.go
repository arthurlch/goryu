package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/scrypt"
)
const (
	MinBcryptCost = 12
	MaxLoginAttempts = 5
	LoginAttemptWindow = 15 * time.Minute
	PasswordResetTokenExpiry = 15 * time.Minute
	EmailVerificationTokenExpiry = 24 * time.Hour
	EmailSendWindow = 1 * time.Minute
	MaxEmailsPerWindow = 3
	SecureTokenLength = 32
)
var (
	PasswordMinLength = 8
	PasswordMaxLength = 128
	passwordHasUpper = regexp.MustCompile(`[A-Z]`)
	passwordHasLower = regexp.MustCompile(`[a-z]`)
	passwordHasDigit = regexp.MustCompile(`\d`)
	passwordHasSpecial = regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`)
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	commonWeakPasswords = map[string]bool{
		"password":    true,
		"123456":      true,
		"password123": true,
		"admin":       true,
		"qwerty":      true,
		"1234567890":  true,
		"password1":   true,
	}
)
type PasswordStrength int
const (
	PasswordWeak PasswordStrength = iota
	PasswordMedium
	PasswordStrong
	PasswordVeryStrong
)
func (ps PasswordStrength) String() string {
	switch ps {
	case PasswordWeak:
		return "weak"
	case PasswordMedium:
		return "medium"
	case PasswordStrong:
		return "strong"
	case PasswordVeryStrong:
		return "very_strong"
	default:
		return "unknown"
	}
}
type SecurePasswordConfig struct {
	MinLength           int
	MaxLength           int
	RequireUpperCase    bool
	RequireLowerCase    bool
	RequireDigits       bool
	RequireSpecialChars bool
	ForbidCommonWords   bool
	ForbidUserInfo      bool
}
func DefaultPasswordConfig() SecurePasswordConfig {
	return SecurePasswordConfig{
		MinLength:           PasswordMinLength,
		MaxLength:           PasswordMaxLength,
		RequireUpperCase:    true,
		RequireLowerCase:    true,
		RequireDigits:       true,
		RequireSpecialChars: true,
		ForbidCommonWords:   true,
		ForbidUserInfo:      true,
	}
}
type PasswordValidationResult struct {
	IsValid    bool
	Strength   PasswordStrength
	Score      int
	Errors     []string
	Warnings   []string
}
func ValidatePassword(password string, config SecurePasswordConfig, userInfo ...string) PasswordValidationResult {
	result := PasswordValidationResult{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
		Score:    0,
	}
	if len(password) < config.MinLength {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Password must be at least %d characters long", config.MinLength))
	} else {
		result.Score += 10
	}
	if len(password) > config.MaxLength {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Password must not exceed %d characters", config.MaxLength))
	}
	if config.RequireUpperCase && !passwordHasUpper.MatchString(password) {
		result.IsValid = false
		result.Errors = append(result.Errors, "Password must contain at least one uppercase letter")
	} else if passwordHasUpper.MatchString(password) {
		result.Score += 10
	}
	if config.RequireLowerCase && !passwordHasLower.MatchString(password) {
		result.IsValid = false
		result.Errors = append(result.Errors, "Password must contain at least one lowercase letter")
	} else if passwordHasLower.MatchString(password) {
		result.Score += 10
	}
	if config.RequireDigits && !passwordHasDigit.MatchString(password) {
		result.IsValid = false
		result.Errors = append(result.Errors, "Password must contain at least one digit")
	} else if passwordHasDigit.MatchString(password) {
		result.Score += 10
	}
	if config.RequireSpecialChars && !passwordHasSpecial.MatchString(password) {
		result.IsValid = false
		result.Errors = append(result.Errors, "Password must contain at least one special character")
	} else if passwordHasSpecial.MatchString(password) {
		result.Score += 15
	}
	if config.ForbidCommonWords {
		lowerPassword := strings.ToLower(password)
		if commonWeakPasswords[lowerPassword] {
			result.IsValid = false
			result.Errors = append(result.Errors, "Password is too common and easily guessable")
		}
		if strings.Contains(lowerPassword, "password") {
			result.IsValid = false
			result.Errors = append(result.Errors, "Password should not contain the word 'password'")
		}
		if strings.Contains(lowerPassword, "123456") {
			result.IsValid = false
			result.Errors = append(result.Errors, "Password should not contain sequential numbers")
		}
	}
	if config.ForbidUserInfo && len(userInfo) > 0 {
		lowerPassword := strings.ToLower(password)
		for _, info := range userInfo {
			if info != "" && len(info) > 2 {
				lowerInfo := strings.ToLower(info)
				if strings.Contains(lowerPassword, lowerInfo) || strings.Contains(lowerInfo, lowerPassword) {
					result.IsValid = false
					result.Errors = append(result.Errors, "Password should not contain personal information")
					break
				}
			}
		}
	}
	entropy := calculatePasswordEntropy(password)
	if entropy > 4.0 {
		result.Score += int(entropy * 5)
	}
	if len(password) >= 12 {
		result.Score += 10
	}
	if len(password) >= 16 {
		result.Score += 10
	}
	result.Strength = calculatePasswordStrength(result.Score, len(password))
	if result.Strength == PasswordMedium {
		result.Warnings = append(result.Warnings, "Consider using a longer password with more character variety")
	}
	return result
}
func calculatePasswordEntropy(password string) float64 {
	if len(password) == 0 {
		return 0
	}
	freq := make(map[rune]int)
	for _, char := range password {
		freq[char]++
	}
	entropy := 0.0
	length := float64(len(password))
	for _, count := range freq {
		p := float64(count) / length
		if p > 0 {
			entropy -= p * (math.Log2(p))
		}
	}
	return entropy
}
func calculatePasswordStrength(score, length int) PasswordStrength {
	if score < 30 || length < 8 {
		return PasswordWeak
	} else if score < 50 || length < 10 {
		return PasswordMedium
	} else if score < 70 || length < 12 {
		return PasswordStrong
	}
	return PasswordVeryStrong
}
func SecureHashPassword(password string, cost int) ([]byte, error) {
	if cost < MinBcryptCost {
		cost = MinBcryptCost
	}
	// Bcrypt handles salting automatically and securely.
	// We do not need to manually manage salts.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}
	return hash, nil
}

func VerifySecurePassword(password string, hashedPassword []byte) bool {
	// Standard bcrypt verification
	return bcrypt.CompareHashAndPassword(hashedPassword, []byte(password)) == nil
}
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if len(email) > 254 {
		return errors.New("email address is too long")
	}
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if strings.ContainsAny(email, "<>\"';&|`$(){}[]\\") {
		return errors.New("email contains invalid characters")
	}
	return nil
}
func GenerateSecureToken() (string, error) {
	token := make([]byte, SecureTokenLength)
	if _, err := rand.Read(token); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %v", err)
	}
	return base64.URLEncoding.EncodeToString(token), nil
}
func GenerateSecureTokenWithExpiry(expiry time.Duration) (string, time.Time, error) {
	token, err := GenerateSecureToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiryTime := time.Now().Add(expiry)
	return token, expiryTime, nil
}
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func GenerateSecureTemporaryPassword(length int) (string, error) {
	if length < 12 {
		length = 12
	}
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits := "0123456789"
	special := "!@#$%^&*"
	allChars := lowercase + uppercase + digits + special
	password := make([]byte, length)
	password[0] = lowercase[randomIndex(len(lowercase))]
	password[1] = uppercase[randomIndex(len(uppercase))]
	password[2] = digits[randomIndex(len(digits))]
	password[3] = special[randomIndex(len(special))]
	for i := 4; i < length; i++ {
		password[i] = allChars[randomIndex(len(allChars))]
	}
	for i := len(password) - 1; i > 0; i-- {
		j := randomIndex(i + 1)
		password[i], password[j] = password[j], password[i]
	}
	return string(password), nil
}
func randomIndex(max int) int {
	if max <= 0 {
		return 0
	}
	randomBytes := make([]byte, 1)
	for {
		rand.Read(randomBytes)
		if int(randomBytes[0]) < (256-(256%max)) {
			return int(randomBytes[0]) % max
		}
	}
}
func GenerateSecureBackupCodes(count int) ([]string, error) {
	if count <= 0 || count > 20 {
		return nil, errors.New("backup code count must be between 1 and 20")
	}
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code1 := make([]byte, 4)
		code2 := make([]byte, 4)
		if _, err := rand.Read(code1); err != nil {
			return nil, fmt.Errorf("failed to generate backup code: %v", err)
		}
		if _, err := rand.Read(code2); err != nil {
			return nil, fmt.Errorf("failed to generate backup code: %v", err)
		}
		codes[i] = fmt.Sprintf("%s-%s", 
			strings.ToUpper(hex.EncodeToString(code1)[:4]),
			strings.ToUpper(hex.EncodeToString(code2)[:4]))
	}
	return codes, nil
}
func DeriveKeyFromPassword(password, salt []byte, keyLen int) ([]byte, error) {
	return scrypt.Key(password, salt, 32768, 8, 1, keyLen)
}
func SecureWipe(data []byte) {
	for i := range data {
		data[i] = 0
	}
}