package auth
import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"github.com/arthurlch/goryu"
	"github.com/golang-jwt/jwt/v5"
)
type AuthService struct {
	jwt         *JWTAuth
	userStore   UserStore
	tokenStore  TokenStore
	rateLimiter *RateLimiter
	emailSender EmailSender
	config      AuthServiceConfig
	logger      Logger
}
type AuthServiceConfig struct {
	AppName                string
	RequireEmailVerification bool
	PasswordConfig         SecurePasswordConfig
	SessionDuration        time.Duration
	RefreshTokenDuration   time.Duration
	EnableRateLimit        bool
	MaxLoginAttempts       int
	RateLimitWindow        time.Duration
	RateLimitBlockDuration time.Duration
	EnableAuditLog         bool
	SecureCookies          bool
	CookieDomain           string
	CookiePath             string
	CSRFProtection         bool
}
func DefaultAuthServiceConfig() AuthServiceConfig {
	return AuthServiceConfig{
		AppName:                  "goryu-app",
		RequireEmailVerification: true,
		PasswordConfig:           DefaultPasswordConfig(),
		SessionDuration:          15 * time.Minute,
		RefreshTokenDuration:     7 * 24 * time.Hour,
		EnableRateLimit:          true,
		MaxLoginAttempts:         5,
		RateLimitWindow:          15 * time.Minute,
		RateLimitBlockDuration:   30 * time.Minute,
		EnableAuditLog:           true,
		SecureCookies:            true,
		CookiePath:               "/",
		CSRFProtection:           true,
	}
}
type EmailSender interface {
	SendVerificationEmail(email, token, verifyURL string) error
	SendPasswordResetEmail(email, token, resetURL string) error
	SendSecurityAlert(email, message string) error
}
type Logger interface {
	LogSecurityEvent(event string, details map[string]interface{})
	LogError(message string, err error)
	LogInfo(message string, details map[string]interface{})
}
type LoginRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required"`
	RememberMe  bool   `json:"remember_me,omitempty"`
	TwoFactorCode string `json:"two_factor_code,omitempty"`
}
type RegisterRequest struct {
	Email     string                 `json:"email" validate:"required,email"`
	Password  string                 `json:"password" validate:"required"`
	FirstName string                 `json:"first_name,omitempty"`
	LastName  string                 `json:"last_name,omitempty"`
	Traits    map[string]interface{} `json:"traits,omitempty"`
}
type PasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}
type PasswordResetConfirmRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required"`
}
type AuthResponse struct {
	Success      bool              `json:"success"`
	Message      string            `json:"message,omitempty"`
	AccessToken  string            `json:"access_token,omitempty"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	ExpiresIn    int64             `json:"expires_in,omitempty"`
	User         *PublicUser       `json:"user,omitempty"`
	RequiresMFA  bool              `json:"requires_mfa,omitempty"`
	Errors       map[string]string `json:"errors,omitempty"`
}
type PublicUser struct {
	ID       string                 `json:"id"`
	Email    string                 `json:"email"`
	Verified bool                   `json:"verified"`
	Traits   map[string]interface{} `json:"traits,omitempty"`
}
func NewAuthService(jwtAuth *JWTAuth, userStore UserStore, tokenStore TokenStore, emailSender EmailSender, config AuthServiceConfig) *AuthService {
	service := &AuthService{
		jwt:         jwtAuth,
		userStore:   userStore,
		tokenStore:  tokenStore,
		emailSender: emailSender,
		config:      config,
	}
	if config.EnableRateLimit {
		service.rateLimiter = NewRateLimiter(
			config.MaxLoginAttempts,
			config.RateLimitWindow,
			config.RateLimitBlockDuration,
		)
	}
	return service
}
func (as *AuthService) SetLogger(logger Logger) {
	as.logger = logger
}
func (as *AuthService) Register(c *goryu.Ctx, req RegisterRequest) AuthResponse {
	if err := ValidateEmail(req.Email); err != nil {
		as.logSecurityEvent("registration_failed", map[string]interface{}{
			"email": req.Email,
			"reason": "invalid_email",
			"ip": as.getClientIP(c),
		})
		return AuthResponse{
			Success: false,
			Message: "Invalid email format",
			Errors:  map[string]string{"email": err.Error()},
		}
	}
	userInfo := []string{req.Email, req.FirstName, req.LastName}
	passwordResult := ValidatePassword(req.Password, as.config.PasswordConfig, userInfo...)
	if !passwordResult.IsValid {
		as.logSecurityEvent("registration_failed", map[string]interface{}{
			"email": req.Email,
			"reason": "weak_password",
			"ip": as.getClientIP(c),
		})
		return AuthResponse{
			Success: false,
			Message: "Password does not meet security requirements",
			Errors:  map[string]string{"password": strings.Join(passwordResult.Errors, "; ")},
		}
	}
	if _, exists := as.userStore.GetUserByEmail(req.Email); exists {
		as.logSecurityEvent("registration_failed", map[string]interface{}{
			"email": req.Email,
			"reason": "email_exists",
			"ip": as.getClientIP(c),
		})
		return AuthResponse{
			Success: false,
			Message: "An account with this email already exists",
			Errors:  map[string]string{"email": "Email already registered"},
		}
	}
	user, err := as.userStore.AddUser(req.Email, req.Password, req.Traits)
	if err != nil {
		as.logError("Failed to create user", err)
		return AuthResponse{
			Success: false,
			Message: "Failed to create account. Please try again.",
		}
	}
	if as.config.RequireEmailVerification {
		verificationToken, _, err := as.jwt.CreateVerificationToken(req.Email)
		if err != nil {
			as.logError("Failed to create verification token", err)
			return AuthResponse{
				Success: false,
				Message: "Account created but failed to send verification email",
			}
		}
		as.tokenStore.AddToken(verificationToken)
		verifyURL := fmt.Sprintf("/auth/verify-email?token=%s", verificationToken)
		if err := as.emailSender.SendVerificationEmail(req.Email, verificationToken, verifyURL); err != nil {
			as.logError("Failed to send verification email", err)
		}
	}
	as.logSecurityEvent("user_registered", map[string]interface{}{
		"user_id": user.ID,
		"email": req.Email,
		"ip": as.getClientIP(c),
		"requires_verification": as.config.RequireEmailVerification,
	})
	message := "Account created successfully"
	if as.config.RequireEmailVerification {
		message += ". Please check your email to verify your account."
	}
	return AuthResponse{
		Success: true,
		Message: message,
		User:    as.toPublicUser(user),
	}
}
func (as *AuthService) Login(c *goryu.Ctx, req LoginRequest) AuthResponse {
	clientIP := as.getClientIP(c)
	if as.config.EnableRateLimit {
		if allowed, resetTime := as.rateLimiter.CheckIPLimit(clientIP); !allowed {
			as.logSecurityEvent("login_rate_limited", map[string]interface{}{
				"ip": clientIP,
				"email": req.Email,
				"reset_time": resetTime,
			})
			return AuthResponse{
				Success: false,
				Message: fmt.Sprintf("Too many login attempts. Try again after %v", resetTime.Format("15:04")),
			}
		}
		if allowed, resetTime := as.rateLimiter.CheckEmailLimit(req.Email); !allowed {
			as.logSecurityEvent("login_rate_limited", map[string]interface{}{
				"ip": clientIP,
				"email": req.Email,
				"reset_time": resetTime,
			})
			return AuthResponse{
				Success: false,
				Message: fmt.Sprintf("Too many login attempts for this email. Try again after %v", resetTime.Format("15:04")),
			}
		}
	}
	if err := ValidateEmail(req.Email); err != nil {
		if as.config.EnableRateLimit {
			as.rateLimiter.RecordFailedAttempt(clientIP, req.Email)
		}
		return AuthResponse{
			Success: false,
			Message: "Invalid email format",
			Errors:  map[string]string{"email": err.Error()},
		}
	}
	user, exists := as.userStore.GetUserByEmail(req.Email)
	if !exists {
		if as.config.EnableRateLimit {
			as.rateLimiter.RecordFailedAttempt(clientIP, req.Email)
		}
		as.logSecurityEvent("login_failed", map[string]interface{}{
			"email": req.Email,
			"reason": "user_not_found",
			"ip": clientIP,
		})
		return AuthResponse{
			Success: false,
			Message: "Invalid email or password",
		}
	}
	if !VerifySecurePassword(req.Password, user.Password) {
		if as.config.EnableRateLimit {
			as.rateLimiter.RecordFailedAttempt(clientIP, req.Email)
		}
		as.logSecurityEvent("login_failed", map[string]interface{}{
			"user_id": user.ID,
			"email": req.Email,
			"reason": "invalid_password",
			"ip": clientIP,
		})
		return AuthResponse{
			Success: false,
			Message: "Invalid email or password",
		}
	}
	if as.config.RequireEmailVerification && !user.Verified {
		as.logSecurityEvent("login_failed", map[string]interface{}{
			"user_id": user.ID,
			"email": req.Email,
			"reason": "email_not_verified",
			"ip": clientIP,
		})
		return AuthResponse{
			Success: false,
			Message: "Please verify your email address before logging in",
		}
	}
	accessToken, err := as.jwt.CreateAuthToken(user.ID)
	if err != nil {
		as.logError("Failed to create access token", err)
		return AuthResponse{
			Success: false,
			Message: "Login failed. Please try again.",
		}
	}
	refreshToken, refreshJTI, err := as.jwt.CreateRefreshToken(user.ID)
	if err != nil {
		as.logError("Failed to create refresh token", err)
		return AuthResponse{
			Success: false,
			Message: "Login failed. Please try again.",
		}
	}
	as.tokenStore.AddToken(refreshJTI)
	if as.config.EnableRateLimit {
		as.rateLimiter.RecordSuccessfulAttempt(req.Email)
	}
	as.setSecureCookies(c, accessToken, refreshToken)
	as.logSecurityEvent("login_success", map[string]interface{}{
		"user_id": user.ID,
		"email": req.Email,
		"ip": clientIP,
		"remember_me": req.RememberMe,
	})
	return AuthResponse{
		Success:      true,
		Message:      "Login successful",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(as.config.SessionDuration.Seconds()),
		User:         as.toPublicUser(user),
	}
}
func (as *AuthService) RequestPasswordReset(c *goryu.Ctx, req PasswordResetRequest) AuthResponse {
	clientIP := as.getClientIP(c)
	if err := ValidateEmail(req.Email); err != nil {
		return AuthResponse{
			Success: false,
			Message: "Invalid email format",
		}
	}
	user, exists := as.userStore.GetUserByEmail(req.Email)
	response := AuthResponse{
		Success: true,
		Message: "If an account with this email exists, you will receive password reset instructions",
	}
	if exists {
		resetToken, resetJTI, err := as.jwt.CreatePasswordResetToken(req.Email)
		if err != nil {
			as.logError("Failed to create password reset token", err)
			return response
		}
		as.tokenStore.AddToken(resetJTI)
		resetURL := fmt.Sprintf("/auth/reset-password?token=%s", resetToken)
		if err := as.emailSender.SendPasswordResetEmail(req.Email, resetToken, resetURL); err != nil {
			as.logError("Failed to send password reset email", err)
		}
		as.logSecurityEvent("password_reset_requested", map[string]interface{}{
			"user_id": user.ID,
			"email": req.Email,
			"ip": clientIP,
		})
	} else {
		as.logSecurityEvent("password_reset_requested_invalid_email", map[string]interface{}{
			"email": req.Email,
			"ip": clientIP,
		})
	}
	return response
}
func (as *AuthService) ConfirmPasswordReset(c *goryu.Ctx, req PasswordResetConfirmRequest) AuthResponse {
	clientIP := as.getClientIP(c)
	token, err := jwt.ParseWithClaims(req.Token, &ResetClaims{}, func(token *jwt.Token) (interface{}, error) {
		return as.jwt.secretKey, nil
	})
	if err != nil || !token.Valid {
		as.logSecurityEvent("password_reset_invalid_token", map[string]interface{}{
			"ip": clientIP,
			"token_error": err.Error(),
		})
		return AuthResponse{
			Success: false,
			Message: "Invalid or expired reset token",
		}
	}
	claims, ok := token.Claims.(*ResetClaims)
	if !ok {
		return AuthResponse{
			Success: false,
			Message: "Invalid token format",
		}
	}
	if !as.tokenStore.UseToken(claims.ID) {
		as.logSecurityEvent("password_reset_token_reuse", map[string]interface{}{
			"email": claims.Subject,
			"ip": clientIP,
		})
		return AuthResponse{
			Success: false,
			Message: "Reset token has already been used",
		}
	}
	user, exists := as.userStore.GetUserByEmail(claims.Subject)
	if !exists {
		as.logSecurityEvent("password_reset_user_not_found", map[string]interface{}{
			"email": claims.Subject,
			"ip": clientIP,
		})
		return AuthResponse{
			Success: false,
			Message: "User not found",
		}
	}
	userInfo := []string{user.Email}
	passwordResult := ValidatePassword(req.NewPassword, as.config.PasswordConfig, userInfo...)
	if !passwordResult.IsValid {
		return AuthResponse{
			Success: false,
			Message: "Password does not meet security requirements",
			Errors:  map[string]string{"password": strings.Join(passwordResult.Errors, "; ")},
		}
	}
	if err := as.userStore.UpdatePassword(user.Email, req.NewPassword); err != nil {
		as.logError("Failed to update password", err)
		return AuthResponse{
			Success: false,
			Message: "Failed to update password. Please try again.",
		}
	}
	if err := as.emailSender.SendSecurityAlert(user.Email, "Your password has been successfully changed"); err != nil {
		as.logError("Failed to send security alert", err)
	}
	as.logSecurityEvent("password_reset_success", map[string]interface{}{
		"user_id": user.ID,
		"email": user.Email,
		"ip": clientIP,
	})
	return AuthResponse{
		Success: true,
		Message: "Password updated successfully",
	}
}
func (as *AuthService) getClientIP(c *goryu.Ctx) string {
	forwarded := c.GetHeader("X-Forwarded-For")
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	realIP := c.GetHeader("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	return c.Request.RemoteAddr
}
func (as *AuthService) setSecureCookies(c *goryu.Ctx, accessToken, refreshToken string) {
	if !as.config.SecureCookies {
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     as.config.CookiePath,
		Domain:   as.config.CookieDomain,
		MaxAge:   int(as.config.SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token", 
		Value:    refreshToken,
		Path:     as.config.CookiePath,
		Domain:   as.config.CookieDomain,
		MaxAge:   int(as.config.RefreshTokenDuration.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
func (as *AuthService) toPublicUser(user *User) *PublicUser {
	return &PublicUser{
		ID:       user.ID,
		Email:    user.Email,
		Verified: user.Verified,
		Traits:   user.Traits,
	}
}
func (as *AuthService) logSecurityEvent(event string, details map[string]interface{}) {
	if as.logger != nil {
		as.logger.LogSecurityEvent(event, details)
	}
}
func (as *AuthService) logError(message string, err error) {
	if as.logger != nil {
		as.logger.LogError(message, err)
	} else {
		log.Printf("AUTH ERROR: %s: %v", message, err)
	}
}
func (as *AuthService) Cleanup() {
	if as.rateLimiter != nil {
		as.rateLimiter.Stop()
	}
}