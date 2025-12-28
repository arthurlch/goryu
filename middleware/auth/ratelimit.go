package auth

import (
	"net"
	"strings"
	"sync"
	"time"
)

type AttemptRecord struct {
	Count        int
	FirstAttempt time.Time
	LastAttempt  time.Time
	Blocked      bool
	BlockedUntil time.Time
}
type RateLimiter struct {
	mu              sync.RWMutex
	ipAttempts      map[string]*AttemptRecord
	emailAttempts   map[string]*AttemptRecord
	maxAttempts     int
	windowDuration  time.Duration
	blockDuration   time.Duration
	cleanupInterval time.Duration
	cleanupTicker   *time.Ticker
	stopCleanup     chan bool
}

func NewRateLimiter(maxAttempts int, windowDuration, blockDuration time.Duration) *RateLimiter {
	rl := &RateLimiter{
		ipAttempts:      make(map[string]*AttemptRecord),
		emailAttempts:   make(map[string]*AttemptRecord),
		maxAttempts:     maxAttempts,
		windowDuration:  windowDuration,
		blockDuration:   blockDuration,
		cleanupInterval: 5 * time.Minute,
		stopCleanup:     make(chan bool),
	}
	rl.startCleanup()
	return rl
}
func (rl *RateLimiter) CheckIPLimit(ip string) (allowed bool, resetTime time.Time) {
	normalizedIP := rl.normalizeIP(ip)
	rl.mu.RLock()
	record, exists := rl.ipAttempts[normalizedIP]
	rl.mu.RUnlock()
	if !exists {
		return true, time.Time{}
	}
	now := time.Now()
	if record.Blocked && now.Before(record.BlockedUntil) {
		return false, record.BlockedUntil
	}
	if now.Sub(record.FirstAttempt) > rl.windowDuration {
		rl.mu.Lock()
		delete(rl.ipAttempts, normalizedIP)
		rl.mu.Unlock()
		return true, time.Time{}
	}
	if record.Count >= rl.maxAttempts {
		rl.mu.Lock()
		record.Blocked = true
		record.BlockedUntil = now.Add(rl.blockDuration)
		rl.mu.Unlock()
		return false, record.BlockedUntil
	}
	return true, time.Time{}
}
func (rl *RateLimiter) CheckEmailLimit(email string) (allowed bool, resetTime time.Time) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	rl.mu.RLock()
	record, exists := rl.emailAttempts[normalizedEmail]
	rl.mu.RUnlock()
	if !exists {
		return true, time.Time{}
	}
	now := time.Now()
	if record.Blocked && now.Before(record.BlockedUntil) {
		return false, record.BlockedUntil
	}
	if now.Sub(record.FirstAttempt) > rl.windowDuration {
		rl.mu.Lock()
		delete(rl.emailAttempts, normalizedEmail)
		rl.mu.Unlock()
		return true, time.Time{}
	}
	if record.Count >= rl.maxAttempts {
		rl.mu.Lock()
		record.Blocked = true
		record.BlockedUntil = now.Add(rl.blockDuration)
		rl.mu.Unlock()
		return false, record.BlockedUntil
	}
	return true, time.Time{}
}
func (rl *RateLimiter) RecordFailedAttempt(ip, email string) {
	now := time.Now()
	normalizedIP := rl.normalizeIP(ip)
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if ipRecord, exists := rl.ipAttempts[normalizedIP]; exists {
		ipRecord.Count++
		ipRecord.LastAttempt = now
	} else {
		rl.ipAttempts[normalizedIP] = &AttemptRecord{
			Count:        1,
			FirstAttempt: now,
			LastAttempt:  now,
		}
	}
	if emailRecord, exists := rl.emailAttempts[normalizedEmail]; exists {
		emailRecord.Count++
		emailRecord.LastAttempt = now
	} else {
		rl.emailAttempts[normalizedEmail] = &AttemptRecord{
			Count:        1,
			FirstAttempt: now,
			LastAttempt:  now,
		}
	}
}
func (rl *RateLimiter) RecordSuccessfulAttempt(email string) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.emailAttempts, normalizedEmail)
}
func (rl *RateLimiter) ResetIPLimit(ip string) {
	normalizedIP := rl.normalizeIP(ip)
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.ipAttempts, normalizedIP)
}
func (rl *RateLimiter) ResetEmailLimit(email string) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.emailAttempts, normalizedEmail)
}
func (rl *RateLimiter) GetAttemptInfo(ip, email string) (ipAttempts, emailAttempts int) {
	normalizedIP := rl.normalizeIP(ip)
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	if ipRecord, exists := rl.ipAttempts[normalizedIP]; exists {
		ipAttempts = ipRecord.Count
	}
	if emailRecord, exists := rl.emailAttempts[normalizedEmail]; exists {
		emailAttempts = emailRecord.Count
	}
	return ipAttempts, emailAttempts
}
func (rl *RateLimiter) normalizeIP(ip string) string {
	if strings.Contains(ip, ",") {
		ip = strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ip
	}
	if parsedIP.To4() == nil {
		ipv6 := parsedIP.To16()
		for i := 8; i < 16; i++ {
			ipv6[i] = 0
		}
		return net.IP(ipv6).String()
	}
	return parsedIP.String()
}
func (rl *RateLimiter) startCleanup() {
	rl.cleanupTicker = time.NewTicker(rl.cleanupInterval)
	go func() {
		for {
			select {
			case <-rl.cleanupTicker.C:
				rl.cleanup()
			case <-rl.stopCleanup:
				rl.cleanupTicker.Stop()
				return
			}
		}
	}()
}
func (rl *RateLimiter) cleanup() {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for ip, record := range rl.ipAttempts {
		expired := now.Sub(record.FirstAttempt) > rl.windowDuration*2
		unblocked := record.Blocked && now.After(record.BlockedUntil)
		if expired || unblocked {
			delete(rl.ipAttempts, ip)
		}
	}
	for email, record := range rl.emailAttempts {
		expired := now.Sub(record.FirstAttempt) > rl.windowDuration*2
		unblocked := record.Blocked && now.After(record.BlockedUntil)
		if expired || unblocked {
			delete(rl.emailAttempts, email)
		}
	}
}
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
}
func (rl *RateLimiter) Stats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	ipBlocked := 0
	emailBlocked := 0
	now := time.Now()
	for _, record := range rl.ipAttempts {
		if record.Blocked && now.Before(record.BlockedUntil) {
			ipBlocked++
		}
	}
	for _, record := range rl.emailAttempts {
		if record.Blocked && now.Before(record.BlockedUntil) {
			emailBlocked++
		}
	}
	return map[string]interface{}{
		"total_ips_tracked":    len(rl.ipAttempts),
		"total_emails_tracked": len(rl.emailAttempts),
		"blocked_ips":          ipBlocked,
		"blocked_emails":       emailBlocked,
		"max_attempts":         rl.maxAttempts,
		"window_duration":      rl.windowDuration,
		"block_duration":       rl.blockDuration,
	}
}
