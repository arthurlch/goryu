package builder

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// one fay I swear Illmove all my types into a type folder, one day

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("config validation error: %s - %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

type Validator struct {
	errors ValidationErrors
}

func NewValidator() *Validator {
	return &Validator{
		errors: make(ValidationErrors, 0),
	}
}

func (v *Validator) Validate(cfg *Config) ValidationErrors {
	v.errors = make(ValidationErrors, 0)

	v.validateApp(&cfg.App)
	v.validateServer(&cfg.Server)
	v.validateRouter(&cfg.Router)
	v.validateStatic(&cfg.Static)
	v.validateSecurity(&cfg.Security)
	v.validateLimits(&cfg.Limits)

	return v.errors
}

func (v *Validator) validateApp(cfg *AppConfig) {
	validEnvironments := []string{"development", "staging", "production", "test"}
	if !contains(validEnvironments, cfg.Environment) {
		v.addError("app.environment", fmt.Sprintf("must be one of: %v", validEnvironments))
	}

	if cfg.Version != "" && !isValidVersion(cfg.Version) {
		v.addError("app.version", "must be a valid semantic version (e.g., 1.0.0)")
	}
}

func (v *Validator) validateServer(cfg *ServerConfig) {
	// validate port
	if cfg.Port < 0 || cfg.Port > 65535 {
		v.addError("server.port", "must be between 0 and 65535")
	}

	// and here host
	if cfg.Host != "" && !isValidHost(cfg.Host) {
		v.addError("server.host", "must be a valid hostname or IP address")
	}

	if cfg.ReadTimeout < 0 {
		v.addError("server.read_timeout", "must be non-negative")
	}
	if cfg.WriteTimeout < 0 {
		v.addError("server.write_timeout", "must be non-negative")
	}
	if cfg.IdleTimeout < 0 {
		v.addError("server.idle_timeout", "must be non-negative")
	}
	if cfg.ShutdownTimeout < 0 {
		v.addError("server.shutdown_timeout", "must be non-negative")
	}

	if cfg.MaxHeaderSize < 0 {
		v.addError("server.max_header_size", "must be non-negative")
	}
	if cfg.MaxHeaderSize > 0 && cfg.MaxHeaderSize < 1024 {
		v.addError("server.max_header_size", "must be at least 1024 bytes if set")
	}

	if cfg.TLS.Enabled {
		if cfg.TLS.CertFile == "" {
			v.addError("server.tls.cert_file", "required when TLS is enabled")
		} else if !fileExists(cfg.TLS.CertFile) {
			v.addError("server.tls.cert_file", "file does not exist")
		}

		if cfg.TLS.KeyFile == "" {
			v.addError("server.tls.key_file", "required when TLS is enabled")
		} else if !fileExists(cfg.TLS.KeyFile) {
			v.addError("server.tls.key_file", "file does not exist")
		}

		validTLSVersions := []string{"TLS1.0", "TLS1.1", "TLS1.2", "TLS1.3"}
		if !contains(validTLSVersions, cfg.TLS.MinVersion) {
			v.addError("server.tls.min_version", fmt.Sprintf("must be one of: %v", validTLSVersions))
		}
	}

	if cfg.ReadTimeout == 0 {
		v.addWarning("server.read_timeout", "consider setting a read timeout to prevent slow client attacks")
	}
	if cfg.WriteTimeout == 0 {
		v.addWarning("server.write_timeout", "consider setting a write timeout to prevent slow client attacks")
	}
}

func (v *Validator) validateRouter(cfg *RouterConfig) {
	// Router config is mostly boolean flags, no specific validation needed!
	// but could've add warnings for certain combinations?
}

func (v *Validator) validateStatic(cfg *StaticConfig) {
	if cfg.Root != "" {
		if !dirExists(cfg.Root) {
			v.addError("static.root", "directory does not exist")
		}
	}

	if cfg.MaxAge < 0 {
		v.addError("static.max_age", "must be non-negative")
	}

	if cfg.CacheDuration < 0 {
		v.addError("static.cache_duration", "must be non-negative")
	}

	if cfg.Index != "" && !isValidFilename(cfg.Index) {
		v.addError("static.index", "must be a valid filename")
	}
}

func (v *Validator) validateSecurity(cfg *SecurityConfig) {
	if cfg.CSRFProtection && cfg.CSRFTokenLength < 16 {
		v.addError("security.csrf_token_length", "must be at least 16 bytes for security")
	}

	validFrameOptions := []string{"DENY", "SAMEORIGIN", "ALLOW-FROM"}
	if cfg.XFrameOptions != "" && !startsWithAny(cfg.XFrameOptions, validFrameOptions) {
		v.addError("security.x_frame_options", fmt.Sprintf("must start with one of: %v", validFrameOptions))
	}

	if cfg.HSTS.Enabled {
		if cfg.HSTS.MaxAge < time.Hour {
			v.addWarning("security.hsts.max_age", "consider setting HSTS max-age to at least 1 hour")
		}

		v.addWarning("security.hsts.enabled", "HSTS requires HTTPS to be effective")
	}

	for i, host := range cfg.AllowedHosts {
		if !isValidHostPattern(host) {
			v.addError(fmt.Sprintf("security.allowed_hosts[%d]", i), "must be a valid hostname or pattern")
		}
	}

	for i, proxy := range cfg.TrustedProxies {
		if !isValidIPOrCIDR(proxy) {
			v.addError(fmt.Sprintf("security.trusted_proxies[%d]", i), "must be a valid IP address or CIDR")
		}
	}
}

func (v *Validator) validateLimits(cfg *LimitsConfig) {
	if cfg.MaxRouteDepth < 1 {
		v.addError("limits.max_route_depth", "must be at least 1")
	}
	if cfg.MaxRouteDepth > 100 {
		v.addWarning("limits.max_route_depth", "very deep routes may impact performance")
	}

	if cfg.MaxTotalRoutes < 1 {
		v.addError("limits.max_total_routes", "must be at least 1")
	}

	if cfg.MaxParametersPerRoute < 0 {
		v.addError("limits.max_parameters_per_route", "must be non-negative")
	}

	if cfg.MaxRequestBodySize < 0 {
		v.addError("limits.max_request_body_size", "must be non-negative")
	}
	if cfg.MaxRequestBodySize > 0 && cfg.MaxRequestBodySize < 1024 {
		v.addWarning("limits.max_request_body_size", "very small body size limit may cause issues")
	}

	if cfg.MaxMultipartMemory < 0 {
		v.addError("limits.max_multipart_memory", "must be non-negative")
	}

	if cfg.MaxConcurrentRequests < 0 {
		v.addError("limits.max_concurrent_requests", "must be non-negative")
	}
}

func (v *Validator) addError(field, message string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

func (v *Validator) addWarning(field, message string) {
	// For now, warnings are just logged, not returned as errors
	//
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func startsWithAny(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func isValidVersion(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

func isValidHost(host string) bool {
	if net.ParseIP(host) != nil {
		return true
	}

	if len(host) == 0 || len(host) > 255 {
		return false
	}

	return true
}

func isValidHostPattern(pattern string) bool {
	// linter is not happy and tells me to use strings.TrimPrefixdefault but I own the linter
	if strings.HasPrefix(pattern, "*.") {
		pattern = pattern[2:]
	}

	return isValidHost(pattern)
}

func isValidIPOrCIDR(s string) bool {
	if net.ParseIP(s) != nil {
		return true
	}

	_, _, err := net.ParseCIDR(s)
	return err == nil
}

func isValidFilename(filename string) bool {
	if filename == "" || strings.ContainsAny(filename, "/\\") {
		return false
	}
	return true
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func ValidateValue(field string, value interface{}) error {
	// placeholder, in case I want to validate single values in future, will never come bacl to that line tho
	return nil
}
