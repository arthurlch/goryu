package builder

import (
	"fmt"
	"time"
)

type Config struct {
	App      AppConfig      `json:"app" yaml:"app"`
	Server   ServerConfig   `json:"server" yaml:"server"`
	Router   RouterConfig   `json:"router" yaml:"router"`
	Static   StaticConfig   `json:"static" yaml:"static"`
	Security SecurityConfig `json:"security" yaml:"security"`
	Limits   LimitsConfig   `json:"limits" yaml:"limits"`
}

type AppConfig struct {
	Name                  string `json:"name" yaml:"name"`
	Version               string `json:"version" yaml:"version"`
	Environment           string `json:"environment" yaml:"environment"`
	DisableStartupMessage bool   `json:"disable_startup_message" yaml:"disable_startup_message"`
	ServerHeader          string `json:"server_header" yaml:"server_header"`
}

type ServerConfig struct {
	Port             int           `json:"port" yaml:"port"`
	Host             string        `json:"host" yaml:"host"`
	ReadTimeout      time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout     time.Duration `json:"write_timeout" yaml:"write_timeout"`
	IdleTimeout      time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
	ShutdownTimeout  time.Duration `json:"shutdown_timeout" yaml:"shutdown_timeout"`
	MaxHeaderSize    int           `json:"max_header_size" yaml:"max_header_size"`
	DisableKeepalive bool          `json:"disable_keepalive" yaml:"disable_keepalive"`
	TLS              TLSConfig     `json:"tls" yaml:"tls"`
}

type TLSConfig struct {
	Enabled    bool   `json:"enabled" yaml:"enabled"`
	CertFile   string `json:"cert_file" yaml:"cert_file"`
	KeyFile    string `json:"key_file" yaml:"key_file"`
	MinVersion string `json:"min_version" yaml:"min_version"`
}

type RouterConfig struct {
	StrictRouting          bool `json:"strict_routing" yaml:"strict_routing"`
	CaseSensitive          bool `json:"case_sensitive" yaml:"case_sensitive"`
	RedirectTrailingSlash  bool `json:"redirect_trailing_slash" yaml:"redirect_trailing_slash"`
	RedirectFixedPath      bool `json:"redirect_fixed_path" yaml:"redirect_fixed_path"`
	HandleMethodNotAllowed bool `json:"handle_method_not_allowed" yaml:"handle_method_not_allowed"`
	HandleOPTIONS          bool `json:"handle_options" yaml:"handle_options"`
	EnableHEADFallback     bool `json:"enable_head_fallback" yaml:"enable_head_fallback"`
}

type StaticConfig struct {
	Root          string        `json:"root" yaml:"root"`
	Index         string        `json:"index" yaml:"index"`
	Browse        bool          `json:"browse" yaml:"browse"`
	MaxAge        time.Duration `json:"max_age" yaml:"max_age"`
	Compress      bool          `json:"compress" yaml:"compress"`
	ByteRange     bool          `json:"byte_range" yaml:"byte_range"`
	Download      bool          `json:"download" yaml:"download"`
	CacheDuration time.Duration `json:"cache_duration" yaml:"cache_duration"`
}

type SecurityConfig struct {
	CSRFProtection     bool       `json:"csrf_protection" yaml:"csrf_protection"`
	CSRFTokenLength    int        `json:"csrf_token_length" yaml:"csrf_token_length"`
	XSSProtection      string     `json:"xss_protection" yaml:"xss_protection"`
	ContentTypeNosniff bool       `json:"content_type_nosniff" yaml:"content_type_nosniff"`
	XFrameOptions      string     `json:"x_frame_options" yaml:"x_frame_options"`
	HSTS               HSTSConfig `json:"hsts" yaml:"hsts"`
	AllowedHosts       []string   `json:"allowed_hosts" yaml:"allowed_hosts"`
	TrustedProxies     []string   `json:"trusted_proxies" yaml:"trusted_proxies"`
}

type HSTSConfig struct {
	Enabled           bool          `json:"enabled" yaml:"enabled"`
	MaxAge            time.Duration `json:"max_age" yaml:"max_age"`
	IncludeSubdomains bool          `json:"include_subdomains" yaml:"include_subdomains"`
	Preload           bool          `json:"preload" yaml:"preload"`
}

type LimitsConfig struct {
	MaxRouteDepth         int `json:"max_route_depth" yaml:"max_route_depth"`
	MaxTotalRoutes        int `json:"max_total_routes" yaml:"max_total_routes"`
	MaxParametersPerRoute int `json:"max_parameters_per_route" yaml:"max_parameters_per_route"`
	MaxRequestBodySize    int `json:"max_request_body_size" yaml:"max_request_body_size"`
	MaxMultipartMemory    int `json:"max_multipart_memory" yaml:"max_multipart_memory"`
	MaxConcurrentRequests int `json:"max_concurrent_requests" yaml:"max_concurrent_requests"`
}

func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:                  "",
			Version:               "1.0.0",
			Environment:           "development",
			DisableStartupMessage: false,
			ServerHeader:          "",
		},
		Server: ServerConfig{
			Port:             3000,
			Host:             "",
			ReadTimeout:      30 * time.Second,
			WriteTimeout:     30 * time.Second,
			IdleTimeout:      120 * time.Second,
			ShutdownTimeout:  30 * time.Second,
			MaxHeaderSize:    1 << 20, // 1MB
			DisableKeepalive: false,
			TLS: TLSConfig{
				Enabled:    false,
				MinVersion: "TLS1.2",
			},
		},
		Router: RouterConfig{
			StrictRouting:          false,
			CaseSensitive:          false,
			RedirectTrailingSlash:  true,
			RedirectFixedPath:      false,
			HandleMethodNotAllowed: true,
			HandleOPTIONS:          true,
			EnableHEADFallback:     true,
		},
		Static: StaticConfig{
			Root:          "./public",
			Index:         "index.html",
			Browse:        false,
			MaxAge:        0,
			Compress:      true,
			ByteRange:     true,
			Download:      false,
			CacheDuration: 0,
		},
		Security: SecurityConfig{
			CSRFProtection:     true,
			CSRFTokenLength:    32,
			XSSProtection:      "1; mode=block",
			ContentTypeNosniff: true,
			XFrameOptions:      "DENY",
			HSTS: HSTSConfig{
				Enabled:           false,                  // Disabled by default, enable for production
				MaxAge:            31536000 * time.Second, // 1 year
				IncludeSubdomains: true,
				Preload:           false,
			},
			AllowedHosts:   []string{},
			TrustedProxies: []string{},
		},
		Limits: LimitsConfig{
			MaxRouteDepth:         32,
			MaxTotalRoutes:        10000,
			MaxParametersPerRoute: 32,
			MaxRequestBodySize:    4 << 20,  // 4MB
			MaxMultipartMemory:    32 << 20, // 32MB
			MaxConcurrentRequests: 0,        // 0 = unlimited power
		},
	}
}

func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	if other.App.Name != "" {
		c.App.Name = other.App.Name
	}
	if other.App.Version != "" {
		c.App.Version = other.App.Version
	}
	if other.App.Environment != "" {
		c.App.Environment = other.App.Environment
	}
	if other.App.ServerHeader != "" {
		c.App.ServerHeader = other.App.ServerHeader
	}
	c.App.DisableStartupMessage = other.App.DisableStartupMessage

	if other.Server.Port != 0 {
		c.Server.Port = other.Server.Port
	}
	if other.Server.Host != "" {
		c.Server.Host = other.Server.Host
	}
	if other.Server.ReadTimeout != 0 {
		c.Server.ReadTimeout = other.Server.ReadTimeout
	}
	if other.Server.WriteTimeout != 0 {
		c.Server.WriteTimeout = other.Server.WriteTimeout
	}
	if other.Server.IdleTimeout != 0 {
		c.Server.IdleTimeout = other.Server.IdleTimeout
	}
	if other.Server.ShutdownTimeout != 0 {
		c.Server.ShutdownTimeout = other.Server.ShutdownTimeout
	}
	if other.Server.MaxHeaderSize != 0 {
		c.Server.MaxHeaderSize = other.Server.MaxHeaderSize
	}

	// ... gotta merging other sections
}

func (c *Config) Clone() *Config {
	clone := *c

	clone.Security.AllowedHosts = append([]string(nil), c.Security.AllowedHosts...)
	clone.Security.TrustedProxies = append([]string(nil), c.Security.TrustedProxies...)

	return &clone
}

func (c *Config) String() string {
	return fmt.Sprintf(
		"Config{\n  App: %+v\n  Server: %+v\n  Router: %+v\n  Static: %+v\n  Security: %+v\n  Limits: %+v\n}",
		c.App, c.Server, c.Router, c.Static, c.Security, c.Limits,
	)
}
