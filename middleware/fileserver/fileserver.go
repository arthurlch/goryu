package fileserver

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)
type Config struct {
	base.BaseConfig
	Root string
	PathPrefix string
	Browse bool
	Index []string
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.Root == "" {
		c.Root = "."
	}
	if c.PathPrefix == "" {
		c.PathPrefix = "/"
	}
	if !strings.HasPrefix(c.PathPrefix, "/") {
		c.PathPrefix = "/" + c.PathPrefix
	}
	if !strings.HasSuffix(c.PathPrefix, "/") {
		c.PathPrefix = c.PathPrefix + "/"
	}
	if len(c.Index) == 0 {
		c.Index = []string{"index.html", "index.htm"}
	}
	return nil
}
func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "Fileserver")
			}
		}
	}
	cleanRoot := filepath.Clean(cfg.Root)
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			if !strings.HasPrefix(c.Request.URL.Path, cfg.PathPrefix) {
				next(c)
				return
			}
			urlPath := strings.TrimPrefix(c.Request.URL.Path, cfg.PathPrefix)
			urlPath = strings.TrimPrefix(urlPath, "/")
			cleanFullPath, err := validateAndSanitizeFilePath(cleanRoot, urlPath)
			if err != nil {
				if cfg.Logger != nil {
					cfg.Logger.Printf("Fileserver security violation: %s (path: %s)", err.Error(), c.Request.URL.Path)
				}
				c.Status(http.StatusForbidden).Text(http.StatusForbidden, "Forbidden")
				return
			}
			info, err := os.Stat(cleanFullPath)
			if err != nil {
				if os.IsNotExist(err) {
					next(c) 
					return
				}
				base.DefaultErrorHandler(c, base.MiddlewareError{
					Middleware: "Fileserver",
					Err:        err,
					StatusCode: http.StatusInternalServerError,
				}, "Fileserver")
				return
			}
			if info.IsDir() {
				if !cfg.Browse {
					for _, indexFile := range cfg.Index {
						indexPath := filepath.Join(cleanFullPath, indexFile)
						if indexInfo, indexErr := os.Stat(indexPath); indexErr == nil && !indexInfo.IsDir() {
							http.ServeFile(c.Writer, c.Request, indexPath)
							return
						}
					}
				}
				if cfg.Browse {
					http.ServeFile(c.Writer, c.Request, cleanFullPath)
					return
				}
				next(c)
				return
			}
			http.ServeFile(c.Writer, c.Request, cleanFullPath)
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}
func WithRoot(root string) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{Root: root})
}
func WithPrefix(prefix, root string) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{PathPrefix: prefix, Root: root})
}
func WithBrowse(root string, browse bool) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{Root: root, Browse: browse})
}
func validateAndSanitizeFilePath(root, requestPath string) (string, error) {
	decodedPath, err := url.QueryUnescape(requestPath)
	if err != nil {
		return "", fmt.Errorf("invalid URL encoding")
	}
	if containsTraversalAttempt(requestPath, decodedPath) {
		return "", fmt.Errorf("directory traversal attack attempt detected")
	}
	root = filepath.Clean(root)
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("invalid root directory")
	}
	cleanPath := filepath.Clean(decodedPath)
	if cleanPath == "." {
		cleanPath = "/"
	}
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}
	fullPath := filepath.Join(rootAbs, cleanPath)
	fullPathAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("invalid file path")
	}
	if !strings.HasPrefix(fullPathAbs, rootAbs+string(filepath.Separator)) && fullPathAbs != rootAbs {
		return "", fmt.Errorf("directory traversal attack detected")
	}
	suspiciousPatterns := []string{
		"~",            
		"\\",           
		"\x00",         
		"<",            
		">",            
	}
	lowerPath := strings.ToLower(cleanPath)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerPath, pattern) {
			return "", fmt.Errorf("suspicious path pattern detected")
		}
	}
	return fullPathAbs, nil
}
func containsTraversalAttempt(originalPath, decodedPath string) bool {
	traversalPatterns := []string{
		"..",                    
		"%2e%2e",               
		"%252e%252e",           
		"..%2f",                
		"%2e.",                 
		".%2e",                 
		"..\\",                 
		"..%5c",                
		"\u002e\u002e",         
	}
	checkPaths := []string{originalPath, decodedPath}
	for _, path := range checkPaths {
		lowerPath := strings.ToLower(path)
		for _, pattern := range traversalPatterns {
			if strings.Contains(lowerPath, strings.ToLower(pattern)) {
				return true
			}
		}
	}
	return false
}