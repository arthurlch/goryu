package plugins

import (
	"fmt"
	"strings"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/cors"
)

type CORSBuilder struct {
	*BaseBuilder
	config cors.Config
}

func NewCORSBuilder() *CORSBuilder {
	return &CORSBuilder{
		BaseBuilder: NewBaseBuilder("cors"),
		config: cors.Config{
			AllowOrigins:     []string{},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Accept", "Content-Type", "Content-Length", "Authorization"},
			ExposeHeaders:    []string{},
			AllowCredentials: false,
			MaxAge:           43200, // 12 hours in secondo !
		},
	}
}

func (b *CORSBuilder) AllowOrigins(origins ...string) *CORSBuilder {
	b.config.AllowOrigins = origins
	return b
}

func (b *CORSBuilder) AllowOrigin(origin string) *CORSBuilder {
	b.config.AllowOrigins = append(b.config.AllowOrigins, origin)
	return b
}

func (b *CORSBuilder) AllowAllOrigins() *CORSBuilder {
	b.config.AllowOrigins = []string{"*"}
	b.SetMetadata("allow_all_origins", true)
	return b
}

func (b *CORSBuilder) AllowMethods(methods ...string) *CORSBuilder {
	b.config.AllowMethods = methods
	return b
}

func (b *CORSBuilder) AllowMethod(method string) *CORSBuilder {
	b.config.AllowMethods = append(b.config.AllowMethods, strings.ToUpper(method))
	return b
}

func (b *CORSBuilder) AllowHeaders(headers ...string) *CORSBuilder {
	b.config.AllowHeaders = headers
	return b
}

func (b *CORSBuilder) AllowHeader(header string) *CORSBuilder {
	b.config.AllowHeaders = append(b.config.AllowHeaders, header)
	return b
}

func (b *CORSBuilder) ExposeHeaders(headers ...string) *CORSBuilder {
	b.config.ExposeHeaders = headers
	return b
}

func (b *CORSBuilder) ExposeHeader(header string) *CORSBuilder {
	b.config.ExposeHeaders = append(b.config.ExposeHeaders, header)
	return b
}

func (b *CORSBuilder) AllowCredentials(allow bool) *CORSBuilder {
	b.config.AllowCredentials = allow
	return b
}

func (b *CORSBuilder) EnableCredentials() *CORSBuilder {
	b.config.AllowCredentials = true
	return b
}

func (b *CORSBuilder) DisableCredentials() *CORSBuilder {
	b.config.AllowCredentials = false
	return b
}

func (b *CORSBuilder) MaxAge(seconds int) *CORSBuilder {
	b.config.MaxAge = seconds
	return b
}

func (b *CORSBuilder) MaxAgeFromDuration(duration time.Duration) *CORSBuilder {
	b.config.MaxAge = int(duration.Seconds())
	return b
}

func (b *CORSBuilder) Permissive() *CORSBuilder {
	b.config.AllowOrigins = []string{"*"}
	b.config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	b.config.AllowHeaders = []string{"*"}
	b.config.AllowCredentials = false // Cannot be true with wildcard origins
	b.SetMetadata("permissive", true)
	return b
}

func (b *CORSBuilder) Restrictive() *CORSBuilder {
	b.config.AllowOrigins = []string{} // Must be explicitly set
	b.config.AllowMethods = []string{"GET", "POST"}
	b.config.AllowHeaders = []string{"Accept", "Content-Type", "Authorization"}
	b.config.AllowCredentials = false
	b.config.MaxAge = 3600 // 1 hour in seconds
	return b
}

func (b *CORSBuilder) Production() *CORSBuilder {
	b.config.AllowOrigins = []string{} // Must be explicitly configured
	b.config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	b.config.AllowHeaders = []string{"Accept", "Content-Type", "Content-Length", "Authorization"}
	b.config.AllowCredentials = false
	b.config.MaxAge = 86400 // 24 hours in seconds
	return b
}

func (b *CORSBuilder) Development() *CORSBuilder {
	b.config.AllowOrigins = []string{"http://localhost:3000", "http://localhost:8080", "http://127.0.0.1:3000"}
	b.config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	b.config.AllowHeaders = []string{"*"}
	b.config.AllowCredentials = true
	b.config.MaxAge = 3600 // 1 hour in seconds
	return b
}

func (b *CORSBuilder) AddCommonHeaders() *CORSBuilder {
	commonHeaders := []string{
		"Accept", "Content-Type", "Content-Length", "Authorization",
		"X-Requested-With", "X-API-Key", "X-CSRF-Token",
	}
	b.config.AllowHeaders = append(b.config.AllowHeaders, commonHeaders...)
	return b
}

func (b *CORSBuilder) Build() context.Middleware {
	if err := b.Validate(); err != nil {
		panic(fmt.Sprintf("CORS configuration invalid: %v", err))
	}
	return cors.New(b.config)
}

func (b *CORSBuilder) Validate() error {
	b.ClearErrors()
	
	if len(b.config.AllowOrigins) == 0 {
		b.AddError(fmt.Errorf("at least one allowed origin must be specified"))
	}
	
	hasWildcardOrigin := false
	for _, origin := range b.config.AllowOrigins {
		if origin == "*" {
			hasWildcardOrigin = true
			break
		}
	}
	
	if hasWildcardOrigin && b.config.AllowCredentials {
		b.AddError(fmt.Errorf("credentials cannot be enabled with wildcard origins"))
	}
	
	if len(b.config.AllowMethods) == 0 {
		b.AddError(fmt.Errorf("at least one allowed method must be specified"))
	}
	
	if hasWildcardOrigin {
		if allowAll, _ := b.GetMetadata("allow_all_origins"); allowAll != true {
			if permissive, _ := b.GetMetadata("permissive"); permissive != true {
				// This was set to wildcard without explicit intent
				b.AddError(fmt.Errorf("wildcard origins detected - this may be insecure. Use AllowAllOrigins() or Permissive() if intentional"))
			}
		}
	}
	
	if b.config.MaxAge < 0 {
		b.AddError(fmt.Errorf("max age cannot be negative"))
	}
	
	return b.BaseBuilder.Validate()
}

func (b *CORSBuilder) Reset() Builder {
	return NewCORSBuilder()
}

func (b *CORSBuilder) Clone() Builder {
	clone := NewCORSBuilder()
	clone.config = b.config
	
	// ddeep copy slices
	if len(b.config.AllowOrigins) > 0 {
		clone.config.AllowOrigins = make([]string, len(b.config.AllowOrigins))
		copy(clone.config.AllowOrigins, b.config.AllowOrigins)
	}
	if len(b.config.AllowMethods) > 0 {
		clone.config.AllowMethods = make([]string, len(b.config.AllowMethods))
		copy(clone.config.AllowMethods, b.config.AllowMethods)
	}
	if len(b.config.AllowHeaders) > 0 {
		clone.config.AllowHeaders = make([]string, len(b.config.AllowHeaders))
		copy(clone.config.AllowHeaders, b.config.AllowHeaders)
	}
	if len(b.config.ExposeHeaders) > 0 {
		clone.config.ExposeHeaders = make([]string, len(b.config.ExposeHeaders))
		copy(clone.config.ExposeHeaders, b.config.ExposeHeaders)
	}
	
	return clone
}

func init() {
	Register("cors", func() Builder {
		return NewCORSBuilder()
	})
}