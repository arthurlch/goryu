package auth_test
import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/middleware/auth"
	"github.com/golang-jwt/jwt/v5"
)
const testSecret = "abcd1234efgh5678ijkl9012mnop3456qrst7890uvwx1234yz56789012"
func setupTestApp() *goryu.App {
	app := goryu.New()
	_, _ = auth.SetupAuthMiddleware(app, testSecret)
	return app
}
func createTestAuthService() *auth.AuthService {
	jwtAuth := auth.NewJWTAuth(testSecret, "test-app")
	userStore := auth.NewInMemoryUserStore()
	tokenStore := auth.NewInMemoryTokenStore()
	emailSender := auth.NewMockEmailSender()
	config := auth.DefaultAuthServiceConfig()
	service := auth.NewAuthService(jwtAuth, userStore, tokenStore, emailSender, config)
	logger := auth.NewSimpleLogger()
	service.SetLogger(logger)
	return service
}
func createTestToken(userID string, secret string, expiresAt time.Time) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
func TestAuthService(t *testing.T) {
	service := createTestAuthService()
	defer service.Cleanup()
	t.Run("UserRegistration", func(t *testing.T) {
		app := setupTestApp()
		payload := map[string]interface{}{
			"email":    "test@example.com",
			"password": "MySecure123!",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Errorf("Expected status 201 for successful registration, got %d", rr.Code)
		}
		var response auth.AuthResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if !response.Success {
			t.Errorf("Expected successful registration, got: %s", response.Message)
		}
	})
	t.Run("UserLogin", func(t *testing.T) {
		app := setupTestApp()
		regPayload := map[string]interface{}{
			"email":    "login@example.com",
			"password": "MySecure123!",
		}
		body, _ := json.Marshal(regPayload)
		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)
		loginPayload := map[string]interface{}{
			"email":    "login@example.com",
			"password": "MySecure123!",
		}
		body, _ = json.Marshal(loginPayload)
		req = httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr = httptest.NewRecorder()
		app.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for unverified user, got %d", rr.Code)
		}
	})
}
func TestPasswordValidation(t *testing.T) {
	config := auth.DefaultPasswordConfig()
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{"ValidPassword", "MySecure123!", true},
		{"TooShort", "Short1!", false},
		{"NoUppercase", "lowercase123!", false},
		{"NoLowercase", "UPPERCASE123!", false},
		{"NoDigits", "SecurePassword!", false},
		{"NoSpecialChars", "SecurePassword123", false},
		{"CommonPassword", "password123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.ValidatePassword(tt.password, config)
			if result.IsValid != tt.valid {
				t.Errorf("Expected password %s to be valid=%v, got %v. Errors: %v", 
					tt.password, tt.valid, result.IsValid, result.Errors)
			}
		})
	}
}
func TestRateLimiting(t *testing.T) {
	rl := auth.NewRateLimiter(3, 5*time.Minute, 10*time.Minute)
	defer rl.Stop()
	testIP := "192.168.1.100"
	testEmail := "test@example.com"
	allowed, _ := rl.CheckIPLimit(testIP)
	if !allowed {
		t.Error("Expected initial IP attempt to be allowed")
	}
	for i := 0; i < 3; i++ {
		rl.RecordFailedAttempt(testIP, testEmail)
	}
	allowed, resetTime := rl.CheckIPLimit(testIP)
	if allowed {
		t.Error("Expected IP to be blocked after max attempts")
	}
	if resetTime.IsZero() {
		t.Error("Expected reset time to be set when blocked")
	}
}
func TestJWTTokens(t *testing.T) {
	jwtAuth := auth.NewJWTAuth(testSecret, "test-app")
	t.Run("CreateAndValidateAuthToken", func(t *testing.T) {
		userID := "user-123"
		token, err := jwtAuth.CreateAuthToken(userID)
		if err != nil {
			t.Fatal(err)
		}
		validatedUserID, err := jwtAuth.ValidateAuthToken(token)
		if err != nil {
			t.Fatal(err)
		}
		if validatedUserID != userID {
			t.Errorf("Expected userID %s, got %s", userID, validatedUserID)
		}
	})
	t.Run("CreateAndValidateRefreshToken", func(t *testing.T) {
		userID := "user-456"
		token, jti, err := jwtAuth.CreateRefreshToken(userID)
		if err != nil {
			t.Fatal(err)
		}
		validatedUserID, validatedJTI, err := jwtAuth.ValidateRefreshToken(token)
		if err != nil {
			t.Fatal(err)
		}
		if validatedUserID != userID {
			t.Errorf("Expected userID %s, got %s", userID, validatedUserID)
		}
		if validatedJTI != jti {
			t.Errorf("Expected JTI %s, got %s", jti, validatedJTI)
		}
	})
}
func TestUserStore(t *testing.T) {
	userStore := auth.NewInMemoryUserStore()
	t.Run("AddAndGetUser", func(t *testing.T) {
		email := "store@example.com"
		password := "password123"
		traits := map[string]interface{}{"role": "user"}
		user, err := userStore.AddUser(email, password, traits)
		if err != nil {
			t.Fatal(err)
		}
		fetchedUser, exists := userStore.GetUserByEmail(email)
		if !exists {
			t.Error("Expected user to exist")
		}
		if fetchedUser.Email != email {
			t.Errorf("Expected email %s, got %s", email, fetchedUser.Email)
		}
		fetchedUser, exists = userStore.GetUserByID(user.ID)
		if !exists {
			t.Error("Expected user to exist")
		}
		if fetchedUser.ID != user.ID {
			t.Errorf("Expected ID %s, got %s", user.ID, fetchedUser.ID)
		}
	})
	t.Run("DuplicateEmail", func(t *testing.T) {
		email := "duplicate@example.com"
		_, err := userStore.AddUser(email, "password1", nil)
		if err != nil {
			t.Fatal(err)
		}
		_, err = userStore.AddUser(email, "password2", nil)
		if err == nil {
			t.Error("Expected error for duplicate email")
		}
	})
}
func TestTokenStore(t *testing.T) {
	tokenStore := auth.NewInMemoryTokenStore()
	defer tokenStore.Stop()
	t.Run("AddAndUseToken", func(t *testing.T) {
		jti := "test-token-123"
		err := tokenStore.AddToken(jti)
		if err != nil {
			t.Fatal(err)
		}
		used := tokenStore.UseToken(jti)
		if !used {
			t.Error("Expected token to be usable")
		}
		used = tokenStore.UseToken(jti)
		if used {
			t.Error("Expected token to not be reusable")
		}
		isUsed := tokenStore.IsTokenUsed(jti)
		if !isUsed {
			t.Error("Expected token to be marked as used")
		}
	})
}
func TestSecurityFunctions(t *testing.T) {
	t.Run("PasswordHashing", func(t *testing.T) {
		password := "testpassword123"
		hash, err := auth.SecureHashPassword(password, auth.MinBcryptCost)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Hash length: %d", len(hash))
		result := auth.VerifySecurePassword(password, hash)
		t.Logf("Verification result: %v", result)
		if !result {
			t.Error("Expected password verification to succeed")
		}
		if auth.VerifySecurePassword("wrongpassword", hash) {
			t.Error("Expected password verification to fail")
		}
	})
	t.Run("TokenGeneration", func(t *testing.T) {
		token, err := auth.GenerateSecureToken()
		if err != nil {
			t.Fatal(err)
		}
		if len(token) == 0 {
			t.Error("Expected non-empty token")
		}
		token2, err := auth.GenerateSecureToken()
		if err != nil {
			t.Fatal(err)
		}
		if token == token2 {
			t.Error("Expected different tokens")
		}
	})
	t.Run("EmailValidation", func(t *testing.T) {
		validEmails := []string{
			"test@example.com",
			"user.name@domain.co.uk",
			"user+tag@example.org",
		}
		invalidEmails := []string{
			"",
			"invalid",
			"@example.com",
			"user@",
			"user<script>@example.com",
		}
		for _, email := range validEmails {
			if err := auth.ValidateEmail(email); err != nil {
				t.Errorf("Expected email %s to be valid, got error: %v", email, err)
			}
		}
		for _, email := range invalidEmails {
			if err := auth.ValidateEmail(email); err == nil {
				t.Errorf("Expected email %s to be invalid", email)
			}
		}
	})
}