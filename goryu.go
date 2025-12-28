package goryu

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/arthurlch/goryu/config/builder"
	goryu_context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/internal/json"
	"github.com/arthurlch/goryu/monitoring"
	"github.com/arthurlch/goryu/router"
)

type Ctx = goryu_context.Context
type Context = goryu_context.Context
type Handler = goryu_context.HandlerFunc
type HandlerFunc = goryu_context.HandlerFunc
type Middleware = goryu_context.Middleware

// Map is a shortcut for map[string]interface{}, useful for JSON responses
type Map map[string]interface{}

type Config struct {
	// AppName is the name of the application.
	// Default: ""
	AppName string
	// ServerHeader is the value of the "Server" header.
	// Default: ""
	ServerHeader string
	// ServerPort is the port the server will listen on.
	// Default: 3000
	ServerPort int
	// ServerHost is the host the server will bind to.
	// Default: ""
	ServerHost string
	// StrictRouting enables strict routing.
	// Default: false
	StrictRouting bool
	// CaseSensitive enables case-sensitive routing.
	// Default: false
	CaseSensitive bool
	// DisableStartupMessage disables the startup message.
	// Default: false
	DisableStartupMessage bool
	// RedirectTrailingSlash enables automatic redirection of trailing slashes.
	// Default: true
	RedirectTrailingSlash *bool
	// EnableHEADFallback allows HEAD requests to fall back to GET handlers.
	// Default: true
	EnableHEADFallback *bool
	// MaxMultipartMemory is the maximum amount of memory to use for multipart form parsing.
	// Default: 10MB
	MaxMultipartMemory int64
	// EnableMonitoring enables the built-in monitoring system.
	// Default: true
	EnableMonitoring *bool
	// JSONEngine specifies which JSON library to use: "standard", "sonic"
	// Default: "standard" (use "sonic" for potentially better performance on large payloads)
	JSONEngine string
}

type App struct {
	Router      *router.Router
	server      *http.Server
	serverMu    sync.RWMutex // Protect concurrent access to server
	middlewares []Middleware
	Config      Config
	mountPath   string
	mountedApps map[string]*App     // Track mounted apps for path updates
	config      *builder.Config     // New configuration from builder
	Monitor     *monitoring.Monitor // Integrated monitoring system
}

func New(config ...Config) *App {
	cfg := Config{
		AppName:            "",
		ServerHeader:       "",
		ServerPort:         3000,
		ServerHost:         "",
		MaxMultipartMemory: 10 << 20, // 10 MB default
	}

	if len(config) > 0 {
		cfg = config[0]
	}

	// Configure JSON engine - default to sonic for JSONHeavy performance
	if cfg.JSONEngine == "standard" {
		json.UseStandardJSON()
	} else {
		// Default to sonic for JSONHeavy performance
		json.UseSonicJSON()
	}

	routerConfig := router.RouterConfig{
		RedirectTrailingSlash:  true, // default
		EnableHEADFallback:     true, // default
		HandleMethodNotAllowed: true,
		HandleOPTIONS:          true,
		RedirectFixedPath:      false,
		ErrorMode:              router.RouterErrorModePanic, // Backward compatible default
		MaxRouteDepth:          32,                          // SECURITY: Reasonable default limit
		MaxTotalRoutes:         10000,                       // SECURITY: Prevent memory exhaustion
		MaxParametersPerRoute:  32,                          // SECURITY: Prevent complex route attacks
	}

	if cfg.RedirectTrailingSlash != nil {
		routerConfig.RedirectTrailingSlash = *cfg.RedirectTrailingSlash
	}
	if cfg.EnableHEADFallback != nil {
		routerConfig.EnableHEADFallback = *cfg.EnableHEADFallback
	}

	enableMonitoring := true
	if cfg.EnableMonitoring != nil {
		enableMonitoring = *cfg.EnableMonitoring
	}

	var monitor *monitoring.Monitor
	if enableMonitoring {
		monitor = monitoring.New(monitoring.Config{
			Enabled:        true,
			MaxEvents:      10000,
			HealthInterval: 30 * time.Second,
			MetricsEnabled: true,
			SafeExecute:    true,
		})
	}

	app := &App{
		Router:      router.New(routerConfig),
		middlewares: make([]Middleware, 0),
		Config:      cfg,
		mountedApps: make(map[string]*App),
		Monitor:     monitor,
	}

	if monitor != nil {
		app.Use(monitor.Middleware())

		app.GET("/_health", monitor.HealthHandler())
		app.GET("/_metrics", monitor.MetricsHandler())
		app.GET("/_events", monitor.EventsHandler())
	}

	return app
}

func (app *App) Use(middleware Middleware) {
	app.middlewares = append(app.middlewares, middleware)
}

func (app *App) applyMiddleware(handler Handler) Handler {
	appliedHandler := handler
	for i := len(app.middlewares) - 1; i >= 0; i-- {
		appliedHandler = app.middlewares[i](appliedHandler)
	}
	return appliedHandler
}

func (app *App) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if app.Config.ServerHeader != "" {
		w.Header().Set("Server", app.Config.ServerHeader)
	}

	// Optimization: Only parse multipart when needed.
	// Skip content-type check for GET/HEAD/DELETE requests as they rarely have bodies
	if req.Method != "GET" && req.Method != "HEAD" && req.Method != "DELETE" {
		contentType := req.Header.Get("Content-Type")
		if len(contentType) >= 19 && contentType[:19] == "multipart/form-data" {
			const defaultMaxMemory = 32 << 20 // 32 MB
			maxMemory := app.Config.MaxMultipartMemory
			if maxMemory == 0 {
				maxMemory = defaultMaxMemory
			}
			if err := req.ParseMultipartForm(maxMemory); err != nil {
				http.Error(w, "Form data too large or invalid", http.StatusBadRequest)
				return
			}
		}
		// Skip form parsing for other content types - let Context handle it lazily
	}

	app.Router.ServeHTTP(w, req)
}

func (app *App) GET(path string, handler Handler) *router.Route {
	return app.Router.GET(path, app.applyMiddleware(handler))
}

func (app *App) POST(path string, handler Handler) *router.Route {
	return app.Router.POST(path, app.applyMiddleware(handler))
}

func (app *App) PUT(path string, handler Handler) *router.Route {
	return app.Router.PUT(path, app.applyMiddleware(handler))
}

func (app *App) DELETE(path string, handler Handler) *router.Route {
	return app.Router.DELETE(path, app.applyMiddleware(handler))
}

func (app *App) PATCH(path string, handler Handler) *router.Route {
	return app.Router.PATCH(path, app.applyMiddleware(handler))
}

func (app *App) HEAD(path string, handler Handler) *router.Route {
	return app.Router.HEAD(path, app.applyMiddleware(handler))
}

func (app *App) OPTIONS(path string, handler Handler) *router.Route {
	return app.Router.OPTIONS(path, app.applyMiddleware(handler))
}

func (app *App) ALL(path string, handler Handler) *router.RouteCollection {
	return app.Router.ALL(path, app.applyMiddleware(handler))
}

func (app *App) Group(prefix string, middlewares ...Middleware) *router.Group {
	return app.Router.Group(prefix, middlewares...)
}

func (app *App) Mount(prefix string, subApp *App) {
	subApp.mountPath = app.mountPath + prefix
	app.mountedApps[prefix] = subApp

	app.updateSubAppMountPaths(subApp)

	mountHandler := func(c *Ctx) {
		c.Set("goryu.mount.original_path", c.Request.URL.Path)
		c.Set("goryu.mount.prefix", prefix)
		c.Set("goryu.mount.sub_path", strings.TrimPrefix(c.Request.URL.Path, prefix))

		subRequest := c.Request.Clone(c.Request.Context())
		subRequest.URL = c.Request.URL

		subURL := *c.Request.URL
		subURL.Path = strings.TrimPrefix(c.Request.URL.Path, prefix)
		if subURL.Path == "" {
			subURL.Path = "/"
		}
		subRequest.URL = &subURL

		subContext := goryu_context.NewContext(c.Writer, subRequest)

		for key, value := range c.Keys {
			if !strings.HasPrefix(key, "goryu.mount.") {
				subContext.Set(key, value)
			}
		}

		subApp.Router.ServeHTTP(c.Writer, subRequest)
	}

	routePath := prefix
	if !strings.HasSuffix(routePath, "/") {
		routePath += "/"
	}
	routePath += "*subpath"

	app.ALL(routePath, mountHandler)
}

func GetMountInfo(c *Ctx) (originalPath, prefix, subPath string, isMounted bool) {
	if original, exists := c.Get("goryu.mount.original_path"); exists {
		if pref, exists := c.Get("goryu.mount.prefix"); exists {
			if sub, exists := c.Get("goryu.mount.sub_path"); exists {
				return original.(string), pref.(string), sub.(string), true
			}
		}
	}
	return "", "", "", false
}

func (app *App) Static(prefix, root string, config ...StaticConfig) {
	// Default configuration
	cfg := StaticConfig{
		Browse:        false,
		Index:         "index.html",
		CacheDuration: 24 * time.Hour,
		MaxAge:        86400, // 24 hours in seconds
	}

	if len(config) > 0 {
		cfg = config[0]
		if cfg.Index == "" {
			cfg.Index = "index.html"
		}
		if cfg.CacheDuration == 0 {
			cfg.CacheDuration = 24 * time.Hour
		}
		if cfg.MaxAge == 0 {
			cfg.MaxAge = 86400
		}
	}

	handler := func(c *Ctx) {
		if cfg.CacheDuration > 0 {
			c.SetHeader("Cache-Control", fmt.Sprintf("public, max-age=%d", cfg.MaxAge))
			c.SetHeader("Expires", time.Now().Add(cfg.CacheDuration).Format(http.TimeFormat))
		}

		filepath := c.Param("filepath")
		var requestPath string
		if filepath == "" {
			requestPath = "/"
		} else {
			requestPath = "/" + filepath
		}

		app.serveStaticFile(c, root, requestPath, cfg)
	}

	routePath := prefix
	if !strings.HasSuffix(routePath, "/") {
		routePath += "/"
	}
	routePath += "*filepath"

	app.Router.GET(routePath, handler)
}

func (app *App) serveStaticFile(c *Ctx, root, requestPath string, cfg StaticConfig) {
	// Security: Proper path sanitization to prevent directory traversal attacks
	sanitizedPath, err := sanitizeStaticPath(root, requestPath)
	if err != nil {
		fmt.Printf("Static file security violation attempt: %s (sanitized from: %s)\n", err.Error(), requestPath)
		c.Status(404)
		c.Text(404, "File Not Found")
		return
	}

	fullPath := sanitizedPath

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.Status(404)
			c.Text(404, "File Not Found")
		} else {
			c.Status(500)
			c.Text(500, "Internal Server Error")
		}
		return
	}

	// Handle directories
	if info.IsDir() {
		if cfg.Browse {
			// Enable directory browsing
			app.serveDirListing(c, fullPath, requestPath, cfg)
			return
		} else {
			// Try to serve index file
			indexPath := fullPath
			if !strings.HasSuffix(indexPath, "/") {
				indexPath += "/"
			}
			indexPath += cfg.Index

			if indexInfo, err := os.Stat(indexPath); err == nil && !indexInfo.IsDir() {
				http.ServeFile(c.Writer, c.Request, indexPath)
				return
			}

			// No index file and browsing disabled
			c.Status(403)
			c.Text(403, "Directory browsing is disabled")
			return
		}
	}

	// Serve the file
	http.ServeFile(c.Writer, c.Request, fullPath)
}

// serveDirListing serves a directory listing when browsing is enabled
func (app *App) serveDirListing(c *Ctx, fullPath, requestPath string, cfg StaticConfig) {
	dir, err := os.Open(fullPath)
	if err != nil {
		c.Status(500)
		c.Text(500, "Internal Server Error")
		return
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		c.Status(500)
		c.Text(500, "Internal Server Error")
		return
	}

	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	c.Status(200)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Directory listing for %s</title>
	<style>
		body { font-family: Arial, sans-serif; margin: 40px; }
		h1 { color: #333; }
		ul { list-style: none; padding: 0; }
		li { margin: 5px 0; }
		a { text-decoration: none; color: #007acc; }
		a:hover { text-decoration: underline; }
		.dir { font-weight: bold; }
		.file { font-weight: normal; }
	</style>
</head>
<body>
	<h1>Directory listing for %s</h1>
	<ul>`, requestPath, requestPath)

	if requestPath != "/" {
		html += `<li><a href="../" class="dir">📁 ..</a></li>`
	}

	for _, file := range files {
		name := file.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		escapedName := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(name, "&", "&amp;"), "<", "&lt;"), ">", "&gt;")

		if file.IsDir() {
			html += fmt.Sprintf(`<li><a href="%s/" class="dir">📁 %s/</a></li>`, escapedName, escapedName)
		} else {
			if cfg.ShowFileSize {
				html += fmt.Sprintf(`<li><a href="%s" class="file">📄 %s</a> <small>(%d bytes)</small></li>`,
					escapedName, escapedName, file.Size())
			} else {
				html += fmt.Sprintf(`<li><a href="%s" class="file">📄 %s</a></li>`,
					escapedName, escapedName)
			}
		}
	}

	html += `</ul></body></html>`
	c.Writer.Write([]byte(html))
}

type StaticConfig struct {
	Browse        bool          `json:"browse"`
	ShowFileSize  bool          `json:"showFileSize"` // SECURITY: Option to hide file sizes in listings
	Index         string        `json:"index"`
	CacheDuration time.Duration `json:"cache_duration"`
	MaxAge        int           `json:"max_age"`
}

func sanitizeStaticPath(root, requestPath string) (string, error) {
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
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}

	fullPath := filepath.Join(rootAbs, cleanPath)
	fullPathAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("invalid file path")
	}

	// This prevents directory traversal attacks including symlink attacks
	if !strings.HasPrefix(fullPathAbs, rootAbs+string(filepath.Separator)) && fullPathAbs != rootAbs {
		return "", fmt.Errorf("directory traversal attack detected")
	}

	suspiciousPatterns := []string{
		"~",    // Home directory access
		"\\",   // Windows path separators on Unix
		"\x00", // Null bytes
		"<",    // Potential XSS in error messages
		">",    // Potential XSS in error messages
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
		"..",           // Basic traversal
		"%2e%2e",       // URL encoded dots
		"%252e%252e",   // Double URL encoded dots
		"..%2f",        // Mixed encoding
		"%2e.",         // Partial encoding
		".%2e",         // Partial encoding
		"..\\",         // Windows-style traversal
		"..%5c",        // URL encoded backslash
		"\u002e\u002e", // Unicode dots
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

func (app *App) updateSubAppMountPaths(subApp *App) {
	for prefix, mountedApp := range subApp.mountedApps {
		mountedApp.mountPath = subApp.mountPath + prefix
		app.updateSubAppMountPaths(mountedApp)
	}
}

func (app *App) MountPath() string {
	return app.mountPath
}

func (app *App) Run(addr string) error {
	if !app.Config.DisableStartupMessage {
		app.printStartupMessage(addr)
	}
	return http.ListenAndServe(addr, app)
}

func (app *App) Listen(addr string) error {
	if !app.Config.DisableStartupMessage {
		app.printStartupMessage(addr)
	}

	app.serverMu.Lock()
	app.server = &http.Server{Addr: addr, Handler: app}
	server := app.server
	app.serverMu.Unlock()

	return server.ListenAndServe()
}

func (app *App) printStartupMessage(addr string) {
	appName := "Goryu"
	if app.Config.AppName != "" {
		appName = app.Config.AppName
	}

	fmt.Printf("\n🚀 %s is ready! Listening on %s\n", appName, addr)
	fmt.Println("   Use Ctrl+C to stop")
	fmt.Println()

	// Print Route Table
	routes := app.Router.Routes()
	if len(routes) > 0 {
		fmt.Println("📍 Registered Routes:")
		fmt.Printf("   %-8s %-30s %s\n", "METHOD", "PATH", "NAME")
		fmt.Println("   " + strings.Repeat("-", 50))

		for _, r := range routes {
			name := r.Name
			if name == "" {
				name = "-"
			}

			// Colorize methods if possible (basic ANSI)
			method := r.Method
			switch method {
			case "GET":
				method = "\033[34mGET\033[0m" // Blue
			case "POST":
				method = "\033[36mPOST\033[0m" // Cyan
			case "PUT":
				method = "\033[33mPUT\033[0m" // Yellow
			case "DELETE":
				method = "\033[31mDELETE\033[0m" // Red
			case "PATCH":
				method = "\033[35mPATCH\033[0m" // Magenta
			}

			fmt.Printf("   %-17s %-30s %s\n", method, r.Path, name)
		}
		fmt.Println()
	}
}

func (app *App) Shutdown() error {
	if app.server == nil {
		return fmt.Errorf("server is not running")
	}
	return app.server.Shutdown(context.Background())
}

func (app *App) ShutdownWithContext(ctx context.Context) error {
	app.serverMu.RLock()
	server := app.server
	app.serverMu.RUnlock()

	if server == nil {
		return fmt.Errorf("server is not running")
	}
	return server.Shutdown(ctx)
}

func (app *App) Server() *http.Server {
	app.serverMu.RLock()
	defer app.serverMu.RUnlock()
	return app.server
}

func (app *App) Handler() http.Handler {
	return app
}

func (app *App) AddHealthCheck(name string, check *monitoring.HealthCheck) {
	if app.Monitor != nil {
		app.Monitor.AddHealthCheck(name, check)
	}
}

func (app *App) RemoveHealthCheck(name string) {
	if app.Monitor != nil {
		app.Monitor.RemoveHealthCheck(name)
	}
}

func (app *App) EmitEvent(eventType monitoring.EventType, message string, data map[string]interface{}) {
	if app.Monitor != nil {
		app.Monitor.EmitEvent(eventType, message, data)
	}
}

func (app *App) GetHealthStatus() monitoring.HealthStatus {
	if app.Monitor != nil {
		return app.Monitor.GetHealthStatus()
	}
	return monitoring.StatusHealthy
}

func (app *App) GetMetrics() *monitoring.Metrics {
	if app.Monitor != nil {
		return app.Monitor.GetMetrics()
	}
	return nil
}

// UseStandardJSON configures Goryu to use the standard library "encoding/json" package.
// This is the default.
func UseStandardJSON() {
	json.UseStandardJSON()
}

// UseSonicJSON configures Goryu to use "github.com/bytedance/sonic" for JSON operations.
// This provides significantly faster performance for complex JSON but may have larger binary size.
func UseSonicJSON() {
	json.UseSonicJSON()
}
