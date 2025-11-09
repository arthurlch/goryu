package session
import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"
	"github.com/arthurlch/goryu/context"
)
type SecurityConfig struct {
	RotateOnLogin          bool
	RotateOnPrivilegeChange bool
	RotationInterval       time.Duration
	BindToIP              bool
	AllowIPChange         bool
	TrustedProxies        []string
	TrackActivity         bool
	IdleTimeout          time.Duration
	AbsoluteTimeout      time.Duration
	DetectAnomalies      bool
	MaxSessionsPerUser   int
	MaxSessionsPerIP     int
}
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		RotateOnLogin:          true,
		RotateOnPrivilegeChange: true,
		RotationInterval:       1 * time.Hour,
		BindToIP:               false, 
		TrackActivity:          true,
		IdleTimeout:            30 * time.Minute,
		AbsoluteTimeout:        24 * time.Hour,
		DetectAnomalies:        true,
		MaxSessionsPerUser:     5,
		MaxSessionsPerIP:       20,
	}
}
func SecureSessionMiddleware(config SecurityConfig) func(next context.HandlerFunc) context.HandlerFunc {
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			session, err := Get(c)
			if err != nil {
				next(c)
				return
			}
			if config.AbsoluteTimeout > 0 {
				if created := session.Get("created_at"); created != nil {
					if createdTime, ok := created.(int64); ok {
						if time.Since(time.Unix(createdTime, 0)) > config.AbsoluteTimeout {
							Destroy(c)
							c.JSON(401, map[string]string{"error": "Session expired"})
							return
						}
					}
				} else {
					session.Set("created_at", time.Now().Unix())
				}
			}
			if config.IdleTimeout > 0 && config.TrackActivity {
				if lastActivity := session.Get("last_activity"); lastActivity != nil {
					if lastActivityTime, ok := lastActivity.(int64); ok {
						if time.Since(time.Unix(lastActivityTime, 0)) > config.IdleTimeout {
							Destroy(c)
							c.JSON(401, map[string]string{"error": "Session expired due to inactivity"})
							return
						}
					}
				}
				session.Set("last_activity", time.Now().Unix())
			}
			if config.BindToIP {
				clientIP := getClientIP(c, config.TrustedProxies)
				if boundIP := session.Get("bound_ip"); boundIP != nil {
					if boundIPStr, ok := boundIP.(string); ok && boundIPStr != clientIP {
						if !config.AllowIPChange {
							Destroy(c)
							c.JSON(401, map[string]string{"error": "Session security violation - IP mismatch"})
							return
						}
						session.Set("ip_changes", append(getIPChanges(session), map[string]interface{}{
							"from": boundIPStr,
							"to":   clientIP,
							"time": time.Now().Unix(),
						}))
					}
				} else {
					session.Set("bound_ip", clientIP)
				}
			}
			if config.RotationInterval > 0 {
				if rotatedAt := session.Get("rotated_at"); rotatedAt != nil {
					if rotatedTime, ok := rotatedAt.(int64); ok {
						if time.Since(time.Unix(rotatedTime, 0)) > config.RotationInterval {
							if err := Regenerate(c); err == nil {
								session.Set("rotated_at", time.Now().Unix())
							}
						}
					}
				} else {
					session.Set("rotated_at", time.Now().Unix())
				}
			}
			next(c)
		}
	}
}
func getClientIP(c *context.Context, trustedProxies []string) string {
	remoteIP, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	if isTrustedProxy(remoteIP, trustedProxies) {
		if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				return strings.TrimSpace(ips[0])
			}
		}
		if xrip := c.GetHeader("X-Real-IP"); xrip != "" {
			return xrip
		}
	}
	return remoteIP
}
func isTrustedProxy(ip string, trustedProxies []string) bool {
	for _, trusted := range trustedProxies {
		if strings.Contains(trusted, "/") {
			_, network, err := net.ParseCIDR(trusted)
			if err == nil && network.Contains(net.ParseIP(ip)) {
				return true
			}
		} else if trusted == ip {
			return true
		}
	}
	return false
}
func getIPChanges(session *Session) []map[string]interface{} {
	if changes := session.Get("ip_changes"); changes != nil {
		if changesList, ok := changes.([]map[string]interface{}); ok {
			return changesList
		}
	}
	return []map[string]interface{}{}
}
type SessionAnomalyDetector struct {
	userSessions map[string][]string 
	ipSessions   map[string][]string 
	config       SecurityConfig
}
func NewSessionAnomalyDetector(config SecurityConfig) *SessionAnomalyDetector {
	return &SessionAnomalyDetector{
		userSessions: make(map[string][]string),
		ipSessions:   make(map[string][]string),
		config:       config,
	}
}
func (sad *SessionAnomalyDetector) CheckAnomaly(userID, sessionID, clientIP string) error {
	if sad.config.MaxSessionsPerUser > 0 {
		sessions := sad.userSessions[userID]
		if len(sessions) >= sad.config.MaxSessionsPerUser {
			return fmt.Errorf("user %s has too many active sessions (%d)", userID, len(sessions))
		}
		sad.userSessions[userID] = append(sessions, sessionID)
	}
	if sad.config.MaxSessionsPerIP > 0 {
		sessions := sad.ipSessions[clientIP]
		if len(sessions) >= sad.config.MaxSessionsPerIP {
			return fmt.Errorf("IP %s has too many active sessions (%d)", clientIP, len(sessions))
		}
		sad.ipSessions[clientIP] = append(sessions, sessionID)
	}
	return nil
}
func (sad *SessionAnomalyDetector) RemoveSession(userID, sessionID, clientIP string) {
	if sessions, exists := sad.userSessions[userID]; exists {
		filtered := make([]string, 0, len(sessions))
		for _, s := range sessions {
			if s != sessionID {
				filtered = append(filtered, s)
			}
		}
		sad.userSessions[userID] = filtered
	}
	if sessions, exists := sad.ipSessions[clientIP]; exists {
		filtered := make([]string, 0, len(sessions))
		for _, s := range sessions {
			if s != sessionID {
				filtered = append(filtered, s)
			}
		}
		sad.ipSessions[clientIP] = filtered
	}
}
func GenerateSessionToken(sessionID string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(sessionID))
	signature := mac.Sum(nil)
	token := sessionID + "." + base64.URLEncoding.EncodeToString(signature)
	return token
}
func ValidateSessionToken(token string, secret []byte) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid token format")
	}
	sessionID := parts[0]
	providedSignature, err := base64.URLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(sessionID))
	expectedSignature := mac.Sum(nil)
	if !hmac.Equal(providedSignature, expectedSignature) {
		return "", fmt.Errorf("invalid token signature")
	}
	return sessionID, nil
}
func OnPrivilegeEscalation(c *context.Context, config SecurityConfig) error {
	if !config.RotateOnPrivilegeChange {
		return nil
	}
	if err := Regenerate(c); err != nil {
		return fmt.Errorf("failed to regenerate session on privilege escalation: %w", err)
	}
	session, err := Get(c)
	if err != nil {
		return err
	}
	session.Set("privilege_escalated_at", time.Now().Unix())
	session.Set("requires_reverification", true)
	return nil
}