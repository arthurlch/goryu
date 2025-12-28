package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/middleware/errors"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandlers struct {
	service *AuthService
}

func NewAuthHandlers(service *AuthService) *AuthHandlers {
	return &AuthHandlers{
		service: service,
	}
}
func (ah *AuthHandlers) RegisterRoutes(app *goryu.App) {
	authGroup := app.Group("/auth")
	authGroup.POST("/register", ah.Register)
	authGroup.POST("/login", ah.Login)
	authGroup.POST("/forgot-password", ah.ForgotPassword)
	authGroup.POST("/reset-password", ah.ResetPassword)
	authGroup.GET("/verify-email", ah.VerifyEmail)
	authGroup.POST("/resend-verification", ah.ResendVerification)
	protected := authGroup.Group("", func(next goryu.Handler) goryu.Handler {
		return func(c *goryu.Ctx) {
			token := ah.extractToken(c)
			if token == "" {
				errors.Error(c).Unauthorized("Authentication required")
				return
			}
			userID, err := ah.validateToken(token)
			if err != nil {
				errors.Error(c).Unauthorized("Invalid or expired token")
				return
			}
			c.Set(UserIDKey, userID)
			next(c)
		}
	})
	protected.POST("/logout", ah.Logout)
	protected.POST("/refresh", ah.RefreshToken)
	protected.POST("/change-password", ah.ChangePassword)
	protected.GET("/profile", ah.GetProfile)
	protected.PUT("/profile", ah.UpdateProfile)
	protected.DELETE("/account", ah.DeleteAccount)
}
func (ah *AuthHandlers) Register(c *goryu.Ctx) {
	var req RegisterRequest
	if err := c.BindJSON(&req); err != nil {
		errors.Error(c).BadRequest("Invalid request format")
		return
	}
	response := ah.service.Register(c, req)
	if response.Success {
		c.JSON(http.StatusCreated, response)
	} else {
		status := http.StatusBadRequest
		if response.Errors != nil {
			if _, exists := response.Errors["email"]; exists && strings.Contains(response.Message, "already exists") {
				status = http.StatusConflict
			}
		}
		c.JSON(status, response)
	}
}
func (ah *AuthHandlers) Login(c *goryu.Ctx) {
	var req LoginRequest
	if err := c.BindJSON(&req); err != nil {
		errors.Error(c).BadRequest("Invalid request format")
		return
	}
	response := ah.service.Login(c, req)
	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		status := http.StatusUnauthorized
		if strings.Contains(response.Message, "Too many") {
			status = http.StatusTooManyRequests
		}
		c.JSON(status, response)
	}
}
func (ah *AuthHandlers) ForgotPassword(c *goryu.Ctx) {
	var req PasswordResetRequest
	if err := c.BindJSON(&req); err != nil {
		errors.Error(c).BadRequest("Invalid request format")
		return
	}
	response := ah.service.RequestPasswordReset(c, req)
	c.JSON(http.StatusOK, response)
}
func (ah *AuthHandlers) ResetPassword(c *goryu.Ctx) {
	var req PasswordResetConfirmRequest
	if err := c.BindJSON(&req); err != nil {
		errors.Error(c).BadRequest("Invalid request format")
		return
	}
	response := ah.service.ConfirmPasswordReset(c, req)
	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		status := http.StatusBadRequest
		if strings.Contains(response.Message, "Invalid or expired") {
			status = http.StatusUnauthorized
		}
		c.JSON(status, response)
	}
}
func (ah *AuthHandlers) VerifyEmail(c *goryu.Ctx) {
	token := c.Query("token")
	if token == "" {
		errors.Error(c).BadRequest("Verification token is required")
		return
	}
	parsedToken, err := jwt.ParseWithClaims(token, &VerificationClaims{}, func(token *jwt.Token) (interface{}, error) {
		return ah.service.jwt.secretKey, nil
	})
	if err != nil || !parsedToken.Valid {
		errors.Error(c).BadRequest("Invalid or expired verification token")
		return
	}
	claims, ok := parsedToken.Claims.(*VerificationClaims)
	if !ok {
		errors.Error(c).BadRequest("Invalid token format")
		return
	}
	if !ah.service.tokenStore.UseToken(claims.ID) {
		errors.Error(c).BadRequest("Verification token has already been used")
		return
	}
	if err := ah.service.userStore.VerifyUserEmail(claims.Subject); err != nil {
		ah.service.logError("Failed to verify user email", err)
		errors.Error(c).Internal(err)
		return
	}
	ah.service.logSecurityEvent("email_verified", map[string]interface{}{
		"email": claims.Subject,
		"ip":    ah.service.getClientIP(c),
	})
	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Email verified successfully",
	})
}
func (ah *AuthHandlers) ResendVerification(c *goryu.Ctx) {
	var req struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := c.BindJSON(&req); err != nil {
		errors.Error(c).BadRequest("Invalid request format")
		return
	}
	user, exists := ah.service.userStore.GetUserByEmail(req.Email)
	if !exists {
		c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "If the email exists and is unverified, a verification email will be sent",
		})
		return
	}
	if user.Verified {
		c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Email is already verified",
		})
		return
	}
	verificationToken, _, err := ah.service.jwt.CreateVerificationToken(req.Email)
	if err != nil {
		ah.service.logError("Failed to create verification token", err)
		errors.Error(c).Internal(err)
		return
	}
	ah.service.tokenStore.AddToken(verificationToken)
	verifyURL := "/auth/verify-email?token=" + verificationToken
	if err := ah.service.emailSender.SendVerificationEmail(req.Email, verificationToken, verifyURL); err != nil {
		ah.service.logError("Failed to send verification email", err)
		errors.Error(c).Internal(err)
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Verification email sent",
	})
}
func (ah *AuthHandlers) Logout(c *goryu.Ctx) {
	refreshToken := ah.extractRefreshToken(c)
	if refreshToken != "" {
		if parsedToken, err := jwt.ParseWithClaims(refreshToken, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
			return ah.service.jwt.secretKey, nil
		}); err == nil {
			if claims, ok := parsedToken.Claims.(*RefreshClaims); ok {
				ah.service.tokenStore.UseToken(claims.ID)
			}
		}
	}
	ah.clearAuthCookies(c)
	userID, _ := c.Get(UserIDKey)
	ah.service.logSecurityEvent("user_logout", map[string]interface{}{
		"user_id": userID,
		"ip":      ah.service.getClientIP(c),
	})
	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}
func (ah *AuthHandlers) RefreshToken(c *goryu.Ctx) {
	refreshToken := ah.extractRefreshToken(c)
	if refreshToken == "" {
		errors.Error(c).Unauthorized("Refresh token required")
		return
	}
	parsedToken, err := jwt.ParseWithClaims(refreshToken, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		return ah.service.jwt.secretKey, nil
	})
	if err != nil || !parsedToken.Valid {
		errors.Error(c).Unauthorized("Invalid or expired refresh token")
		return
	}
	claims, ok := parsedToken.Claims.(*RefreshClaims)
	if !ok {
		errors.Error(c).Unauthorized("Invalid refresh token format")
		return
	}
	if !ah.service.tokenStore.UseToken(claims.ID) {
		errors.Error(c).Unauthorized("Refresh token has already been used")
		return
	}
	newAccessToken, err := ah.service.jwt.CreateAuthToken(claims.Subject)
	if err != nil {
		ah.service.logError("Failed to create new access token", err)
		errors.Error(c).Internal(err)
		return
	}
	newRefreshToken, newRefreshJTI, err := ah.service.jwt.CreateRefreshToken(claims.Subject)
	if err != nil {
		ah.service.logError("Failed to create new refresh token", err)
		errors.Error(c).Internal(err)
		return
	}
	ah.service.tokenStore.AddToken(newRefreshJTI)
	ah.service.setSecureCookies(c, newAccessToken, newRefreshToken)
	c.JSON(http.StatusOK, map[string]interface{}{
		"success":       true,
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    int64(ah.service.config.SessionDuration.Seconds()),
	})
}
func (ah *AuthHandlers) ChangePassword(c *goryu.Ctx) {
	var req struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password" validate:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		errors.Error(c).BadRequest("Invalid request format")
		return
	}
	userIDValue, _ := c.Get(UserIDKey)
	userID := userIDValue.(string)
	user, exists := ah.service.userStore.GetUserByID(userID)
	if !exists {
		errors.Error(c).NotFound("user")
		return
	}
	if !VerifySecurePassword(req.CurrentPassword, user.Password) {
		errors.Error(c).BadRequest("Current password is incorrect")
		return
	}
	userInfo := []string{user.Email}
	passwordResult := ValidatePassword(req.NewPassword, ah.service.config.PasswordConfig, userInfo...)
	if !passwordResult.IsValid {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Password does not meet security requirements",
			"errors":  map[string]string{"password": strings.Join(passwordResult.Errors, "; ")},
		})
		return
	}
	if err := ah.service.userStore.UpdatePassword(user.Email, req.NewPassword); err != nil {
		ah.service.logError("Failed to update password", err)
		errors.Error(c).Internal(err)
		return
	}
	if err := ah.service.emailSender.SendSecurityAlert(user.Email, "Your password has been changed"); err != nil {
		ah.service.logError("Failed to send security alert", err)
	}
	ah.service.logSecurityEvent("password_changed", map[string]interface{}{
		"user_id": userID,
		"ip":      ah.service.getClientIP(c),
	})
	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Password updated successfully",
	})
}
func (ah *AuthHandlers) GetProfile(c *goryu.Ctx) {
	userIDValue, _ := c.Get(UserIDKey)
	userID := userIDValue.(string)
	user, exists := ah.service.userStore.GetUserByID(userID)
	if !exists {
		errors.Error(c).NotFound("user")
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"user":    ah.service.toPublicUser(user),
	})
}
func (ah *AuthHandlers) UpdateProfile(c *goryu.Ctx) {
	var req struct {
		Traits map[string]interface{} `json:"traits"`
	}
	if err := c.BindJSON(&req); err != nil {
		errors.Error(c).BadRequest("Invalid request format")
		return
	}
	userIDValue, _ := c.Get(UserIDKey)
	userID := userIDValue.(string)
	_, exists := ah.service.userStore.GetUserByID(userID)
	if !exists {
		errors.Error(c).NotFound("user")
		return
	}
	ah.service.logSecurityEvent("profile_updated", map[string]interface{}{
		"user_id": userID,
		"ip":      ah.service.getClientIP(c),
	})
	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Profile updated successfully",
	})
}
func (ah *AuthHandlers) DeleteAccount(c *goryu.Ctx) {
	var req struct {
		Password string `json:"password" validate:"required"`
		Confirm  string `json:"confirm" validate:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		errors.Error(c).BadRequest("Invalid request format")
		return
	}
	if req.Confirm != "DELETE" {
		errors.Error(c).BadRequest("Please type 'DELETE' to confirm account deletion")
		return
	}
	userIDValue, _ := c.Get(UserIDKey)
	userID := userIDValue.(string)
	user, exists := ah.service.userStore.GetUserByID(userID)
	if !exists {
		errors.Error(c).NotFound("user")
		return
	}
	if !VerifySecurePassword(req.Password, user.Password) {
		errors.Error(c).BadRequest("Password is incorrect")
		return
	}
	ah.service.logSecurityEvent("account_deleted", map[string]interface{}{
		"user_id": userID,
		"email":   user.Email,
		"ip":      ah.service.getClientIP(c),
	})
	ah.clearAuthCookies(c)
	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Account deleted successfully",
	})
}
func (ah *AuthHandlers) extractToken(c *goryu.Ctx) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return parts[1]
		}
	}
	if cookie, err := c.Cookie("access_token"); err == nil {
		return cookie.Value
	}
	return ""
}
func (ah *AuthHandlers) extractRefreshToken(c *goryu.Ctx) string {
	if cookie, err := c.Cookie("refresh_token"); err == nil {
		return cookie.Value
	}
	var body map[string]string
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err == nil {
		if token, exists := body["refresh_token"]; exists {
			return token
		}
	}
	return ""
}
func (ah *AuthHandlers) validateToken(token string) (string, error) {
	parsedToken, err := jwt.ParseWithClaims(token, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		return ah.service.jwt.secretKey, nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := parsedToken.Claims.(*AuthClaims); ok && parsedToken.Valid {
		return claims.Subject, nil
	}
	return "", fmt.Errorf("invalid token claims")
}
func (ah *AuthHandlers) clearAuthCookies(c *goryu.Ctx) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     ah.service.config.CookiePath,
		Domain:   ah.service.config.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     ah.service.config.CookiePath,
		Domain:   ah.service.config.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
