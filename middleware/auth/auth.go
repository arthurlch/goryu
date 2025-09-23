package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const UserIDKey = "userID"

type User struct {
	ID       string                 `json:"id"`
	Email    string                 `json:"email"`
	Password []byte                 `json:"-"` // Store hashed password, ignore in JSON responses.
	Verified bool                   `json:"verified"`
	Traits   map[string]interface{} `json:"traits,omitempty"`
}

type UserStore interface {
	AddUser(email, password string, traits map[string]interface{}) (*User, error)
	GetUserByEmail(email string) (*User, bool)
	GetUserByID(id string) (*User, bool)
	UpdatePassword(email, newPassword string) error
	VerifyUserEmail(email string) error
}

type TokenStore interface {
	AddToken(jti string)
	UseToken(jti string) bool
}

func HashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func CheckPasswordHash(password string, hash []byte) bool {
	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	return err == nil
}

// JWT

type JWTAuth struct {
	secretKey []byte
	appName   string
}

type (
	AuthClaims         struct{ jwt.RegisteredClaims }
	RefreshClaims      struct{ jwt.RegisteredClaims }
	ResetClaims        struct{ jwt.RegisteredClaims }
	VerificationClaims struct{ jwt.RegisteredClaims }
)

func NewJWTAuth(secretKey, appName string) *JWTAuth {
	return &JWTAuth{
		secretKey: []byte(secretKey),
		appName:   appName,
	}
}

func (ja *JWTAuth) CreateAuthToken(userID string) (string, error) {
	claims := AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Audience:  jwt.ClaimStrings{ja.appName},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    ja.appName,
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(ja.secretKey)
}

func (ja *JWTAuth) CreateRefreshToken(userID string) (string, string, error) {
	jti := uuid.New().String()
	claims := RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Audience:  jwt.ClaimStrings{ja.appName},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    ja.appName,
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(ja.secretKey)
	return signedToken, jti, err
}

func (ja *JWTAuth) CreatePasswordResetToken(email string) (string, string, error) {
	jti := uuid.New().String()
	claims := ResetClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Audience:  jwt.ClaimStrings{ja.appName},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    ja.appName,
			Subject:   email,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(ja.secretKey)
	return signedToken, jti, err
}

func (ja *JWTAuth) CreateVerificationToken(email string) (string, string, error) {
	jti := uuid.New().String()
	claims := VerificationClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Audience:  jwt.ClaimStrings{ja.appName},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    ja.appName,
			Subject:   email,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(ja.secretKey)
	return signedToken, jti, err
}

type Config struct {
	SecretKey string
	Next      func(c *goryu.Context) bool
}

func New(config Config) goryu.Middleware {
	if config.SecretKey == "" {
		// Return a middleware that always returns an error
		return func(next goryu.HandlerFunc) goryu.HandlerFunc {
			return func(c *goryu.Context) {
				_ = c.Status(http.StatusInternalServerError).Text(http.StatusInternalServerError, "Auth middleware error: no secret key configured")
			}
		}
	}

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.SecretKey), nil
	}

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if config.Next != nil && config.Next(c) {
				next(c)
				return
			}

			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				_ = c.JSON(401, map[string]string{"error": "Authorization header required"})
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				_ = c.JSON(401, map[string]string{"error": "Invalid Authorization header format"})
				return
			}

			tokenString := parts[1]
			token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, keyFunc)

			if err != nil {
				if errors.Is(err, jwt.ErrTokenExpired) {
					_ = c.JSON(401, map[string]string{"error": "Token has expired"})
				} else {
					_ = c.JSON(401, map[string]string{"error": "Invalid token"})
				}
				return
			}

			if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
				c.Set(UserIDKey, claims.Subject)
				next(c)
			} else {
				_ = c.JSON(401, map[string]string{"error": "Invalid token claims"})
			}
		}
	}
}
