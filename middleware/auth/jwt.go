package auth
import (
	"errors"
	"time"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)
type JWTAuth struct {
	secretKey         []byte
	issuer           string
	accessExpiry     time.Duration
	refreshExpiry    time.Duration
	verificationExpiry time.Duration
	resetExpiry      time.Duration
}
type AuthClaims struct {
	jwt.RegisteredClaims
}
type RefreshClaims struct {
	jwt.RegisteredClaims
	Type string `json:"type"`
}
type VerificationClaims struct {
	jwt.RegisteredClaims
	Type string `json:"type"`
}
type ResetClaims struct {
	jwt.RegisteredClaims
	Type string `json:"type"`
}
func NewJWTAuth(secretKey, issuer string) *JWTAuth {
	return &JWTAuth{
		secretKey:          []byte(secretKey),
		issuer:            issuer,
		accessExpiry:      15 * time.Minute,
		refreshExpiry:     7 * 24 * time.Hour,
		verificationExpiry: 24 * time.Hour,
		resetExpiry:       15 * time.Minute,
	}
}
func (j *JWTAuth) SetExpiryTimes(access, refresh, verification, reset time.Duration) {
	j.accessExpiry = access
	j.refreshExpiry = refresh
	j.verificationExpiry = verification
	j.resetExpiry = reset
}
func (j *JWTAuth) CreateAuthToken(userID string) (string, error) {
	now := time.Now()
	claims := AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID,
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessExpiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}
func (j *JWTAuth) CreateRefreshToken(userID string) (string, string, error) {
	now := time.Now()
	jti := uuid.New().String()
	claims := RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   userID,
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshExpiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
		Type: "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secretKey)
	return tokenString, jti, err
}
func (j *JWTAuth) CreateVerificationToken(email string) (string, string, error) {
	now := time.Now()
	jti := uuid.New().String()
	claims := VerificationClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   email,
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.verificationExpiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
		Type: "verification",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secretKey)
	return tokenString, jti, err
}
func (j *JWTAuth) CreatePasswordResetToken(email string) (string, string, error) {
	now := time.Now()
	jti := uuid.New().String()
	claims := ResetClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   email,
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.resetExpiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
		Type: "reset",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secretKey)
	return tokenString, jti, err
}
func (j *JWTAuth) ValidateAuthToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return j.secretKey, nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
		return claims.Subject, nil
	}
	return "", errors.New("invalid token claims")
}
func (j *JWTAuth) ValidateRefreshToken(tokenString string) (string, string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return j.secretKey, nil
	})
	if err != nil {
		return "", "", err
	}
	if claims, ok := token.Claims.(*RefreshClaims); ok && token.Valid {
		if claims.Type != "refresh" {
			return "", "", errors.New("invalid token type")
		}
		return claims.Subject, claims.ID, nil
	}
	return "", "", errors.New("invalid token claims")
}
func (j *JWTAuth) ValidateVerificationToken(tokenString string) (string, string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &VerificationClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return j.secretKey, nil
	})
	if err != nil {
		return "", "", err
	}
	if claims, ok := token.Claims.(*VerificationClaims); ok && token.Valid {
		if claims.Type != "verification" {
			return "", "", errors.New("invalid token type")
		}
		return claims.Subject, claims.ID, nil
	}
	return "", "", errors.New("invalid token claims")
}
func (j *JWTAuth) ValidatePasswordResetToken(tokenString string) (string, string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &ResetClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return j.secretKey, nil
	})
	if err != nil {
		return "", "", err
	}
	if claims, ok := token.Claims.(*ResetClaims); ok && token.Valid {
		if claims.Type != "reset" {
			return "", "", errors.New("invalid token type")
		}
		return claims.Subject, claims.ID, nil
	}
	return "", "", errors.New("invalid token claims")
}