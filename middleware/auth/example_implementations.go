package auth

import (
	"fmt"
	"log"
)
type MockEmailSender struct {
	SentEmails []EmailRecord
}
type EmailRecord struct {
	To      string
	Subject string
	Content string
	Type    string
}
func NewMockEmailSender() *MockEmailSender {
	return &MockEmailSender{
		SentEmails: make([]EmailRecord, 0),
	}
}
func (m *MockEmailSender) SendVerificationEmail(email, token, verifyURL string) error {
	m.SentEmails = append(m.SentEmails, EmailRecord{
		To:      email,
		Subject: "Verify Your Email Address",
		Content: fmt.Sprintf("Please verify your email by clicking: %s", verifyURL),
		Type:    "verification",
	})
	log.Printf("MOCK EMAIL: Verification email sent to %s with URL: %s", email, verifyURL)
	return nil
}
func (m *MockEmailSender) SendPasswordResetEmail(email, token, resetURL string) error {
	m.SentEmails = append(m.SentEmails, EmailRecord{
		To:      email,
		Subject: "Reset Your Password",
		Content: fmt.Sprintf("Reset your password by clicking: %s", resetURL),
		Type:    "password_reset",
	})
	log.Printf("MOCK EMAIL: Password reset email sent to %s with URL: %s", email, resetURL)
	return nil
}
func (m *MockEmailSender) SendSecurityAlert(email, message string) error {
	m.SentEmails = append(m.SentEmails, EmailRecord{
		To:      email,
		Subject: "Security Alert",
		Content: message,
		Type:    "security_alert",
	})
	log.Printf("MOCK EMAIL: Security alert sent to %s: %s", email, message)
	return nil
}
func (m *MockEmailSender) GetSentEmails() []EmailRecord {
	return m.SentEmails
}
func (m *MockEmailSender) ClearSentEmails() {
	m.SentEmails = make([]EmailRecord, 0)
}
type SimpleLogger struct {
	EnableSecurityLog bool
	EnableErrorLog    bool
	EnableInfoLog     bool
}
func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{
		EnableSecurityLog: true,
		EnableErrorLog:    true,
		EnableInfoLog:     true,
	}
}
func (l *SimpleLogger) LogSecurityEvent(event string, details map[string]interface{}) {
	if !l.EnableSecurityLog {
		return
	}
	log.Printf("SECURITY EVENT: %s - Details: %+v", event, details)
}
func (l *SimpleLogger) LogError(message string, err error) {
	if !l.EnableErrorLog {
		return
	}
	log.Printf("ERROR: %s - %v", message, err)
}
func (l *SimpleLogger) LogInfo(message string, details map[string]interface{}) {
	if !l.EnableInfoLog {
		return
	}
	log.Printf("INFO: %s - Details: %+v", message, details)
}