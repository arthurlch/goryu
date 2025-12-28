package goryuctx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// SECUCHECK represents security checks implemented in the code
// to prevent common vulnerabilities such as path traversal attacks,
// file upload issues, and header manipulations.
// Overall security posture is enhanced by validating inputs,
// sanitizing file paths, and restricting file operations.
// I believe we reached a good balance between security and usability,
// but continuous testing is essential to maintain security and I wanna improve it even further

func (c *Context) Query(name string) string {
	return c.Request.URL.Query().Get(name)
}

func (c *Context) Form(name string) string {
	return c.Request.FormValue(name)
}

func (c *Context) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return c.Request.FormFile(key)
}

func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dstFilename string) error {
	const uploadDir = "uploads"
	const maxFilenameLength = 255

	// SECUCHECK: Comprehensive filename validation
	if err := validateUploadFilename(dstFilename); err != nil {
		return err
	}

	// SECUCHECK: File size validation (prevent huge file uploads)
	const maxFileSize = 50 << 20 // 50MB
	if file.Size > maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d bytes)", file.Size, maxFileSize)
	}

	// SECUCHECK: Filename length check
	if len(dstFilename) > maxFilenameLength {
		return errors.New("filename too long")
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil { // More restrictive permissions
		return err
	}

	// SECUCHECK: Enhanced path validation to prevent various attack vectors
	cleanFilename, err := validateAndSanitizeUploadPath(dstFilename)
	if err != nil {
		return fmt.Errorf("invalid filename: %w", err)
	}

	safePath := filepath.Join(uploadDir, cleanFilename)

	// SECUCHECK: Double-check the resolved path is within upload directory
	absUploadDir, err := filepath.Abs(uploadDir)
	if err != nil {
		return fmt.Errorf("failed to resolve upload directory: %w", err)
	}

	absSafePath, err := filepath.Abs(safePath)
	if err != nil {
		return fmt.Errorf("failed to resolve destination path: %w", err)
	}

	// SECUCHECK: Comprehensive path traversal protection
	if !strings.HasPrefix(absSafePath, absUploadDir+string(filepath.Separator)) && absSafePath != absUploadDir {
		return errors.New("invalid destination: path traversal detected")
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	// SECUCHECK: Create file with restrictive permissions
	out, err := os.OpenFile(absSafePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	var copyErr, closeErr error
	_, copyErr = io.Copy(out, src)
	closeErr = out.Close()

	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func validateUploadFilename(filename string) error {
	if filename == "" {
		return errors.New("filename cannot be empty")
	}

	// SECURITY: Check for path separators (both Unix and Windows)
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return errors.New("invalid destination filename: contains path separators")
	}

	// SECUCHECK: Check for hidden files and current/parent directory references
	if strings.HasPrefix(filename, ".") {
		return errors.New("invalid destination filename: hidden files not allowed")
	}

	// SECUCHECK: Check for null bytes and other dangerous characters
	dangerousChars := []string{"\x00", "<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range dangerousChars {
		if strings.Contains(filename, char) {
			return fmt.Errorf("invalid destination filename: contains dangerous character '%s'", char)
		}
	}

	// SECUCHECK: Check for reserved names (Windows)
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	filenameUpper := strings.ToUpper(filename)
	baseNameUpper := strings.ToUpper(strings.Split(filename, ".")[0])

	for _, reserved := range reservedNames {
		if filenameUpper == reserved || baseNameUpper == reserved {
			return fmt.Errorf("invalid destination filename: '%s' is a reserved name", filename)
		}
	}

	// SECUCHECK: Check for excessively long extensions
	if strings.Contains(filename, ".") {
		parts := strings.Split(filename, ".")
		if len(parts) > 2 {
			return errors.New("invalid destination filename: multiple extensions not allowed")
		}
		extension := parts[len(parts)-1]
		if len(extension) > 10 {
			return errors.New("invalid destination filename: extension too long")
		}
	}

	return nil
}

// SECUCHECK: Protects against various path traversal and attack vectors
func validateAndSanitizeUploadPath(filename string) (string, error) {
	if filename == "" {
		return "", errors.New("filename cannot be empty")
	}

	// SECUCHECK: Check filename length to prevent long path attacks
	if len(filename) > 255 {
		return "", errors.New("filename too long")
	}

	// SECUCHECK: Validate UTF-8 encoding
	if !utf8.ValidString(filename) {
		return "", errors.New("filename contains invalid UTF-8 characters")
	}

	// SECUCHECK: Normalize Unicode to prevent normalization attacks
	normalized := norm.NFC.String(filename)

	// SECUCHECK: Check for directory traversal patterns
	traversalPatterns := []string{
		"..",             // Basic traversal
		"%2e%2e",         // URL encoded dots
		"%252e%252e",     // Double URL encoded dots
		"..%2f",          // Mixed encoding
		"%2e.",           // Partial encoding
		".%2e",           // Partial encoding
		"..\\",           // Windows-style traversal
		"..%5c",          // URL encoded backslash
		"\\u002e\\u002e", // Unicode dots
		"\u002e\u002e",   // Unicode path separators
		"\u2024",         // One dot leader (Unicode)
		"\uFF0E",         // Fullwidth full stop
	}

	lowerFilename := strings.ToLower(normalized)
	for _, pattern := range traversalPatterns {
		if strings.Contains(lowerFilename, strings.ToLower(pattern)) {
			return "", errors.New("filename contains path traversal patterns")
		}
	}

	// SECURITY: Check for suspicious characters
	suspiciousChars := []string{
		"\x00", // Null byte
		"\r",   // Carriage return
		"\n",   // Newline
		"\t",   // Tab
		"<",    // HTML/XML
		">",    // HTML/XML
		":",    // Drive separator (Windows)
		"|",    // Pipe character
		"?",    // Wildcard
		"*",    // Wildcard
		"\"",   // Quote
	}

	for _, char := range suspiciousChars {
		if strings.Contains(normalized, char) {
			return "", fmt.Errorf("filename contains suspicious character: %s", char)
		}
	}

	// SECUCHECK: Check for reserved Windows filenames
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	baseFilename := strings.TrimSuffix(normalized, filepath.Ext(normalized))
	for _, reserved := range reservedNames {
		if strings.EqualFold(baseFilename, reserved) {
			return "", fmt.Errorf("filename uses reserved name: %s", reserved)
		}
	}

	// SECUCHECK: Check for files starting with dot (hidden files)
	if strings.HasPrefix(normalized, ".") {
		return "", errors.New("hidden files (starting with '.') are not allowed")
	}

	// SECUCHECK: Check for executable file extensions (configurable based on needs)
	dangerousExtensions := []string{
		".exe", ".bat", ".cmd", ".com", ".pif", ".scr", ".vbs", ".js",
		".jar", ".sh", ".bin", ".app", ".deb", ".dmg", ".pkg", ".msi",
		".php", ".asp", ".aspx", ".jsp", ".pl", ".py", ".rb",
	}

	ext := strings.ToLower(filepath.Ext(normalized))
	for _, dangerous := range dangerousExtensions {
		if ext == dangerous {
			return "", fmt.Errorf("file extension '%s' is not allowed for security reasons", ext)
		}
	}

	// SECUCHECK: Final cleanup - use filepath.Clean for normalization
	cleaned := filepath.Clean(normalized)

	// SECUCHECK: Ensure cleaned path doesn't escape current directory
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") {
		return "", errors.New("path attempts to escape upload directory")
	}

	return cleaned, nil
}

func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

func (c *Context) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

// SECUCHECK: Only trusts proxy headers if explicitly configured with trusted proxies.
// By default, only uses the direct connection IP to prevent spoofing attacks.
func (c *Context) RemoteIP() string {
	directIP, _, _ := net.SplitHostPort(c.Request.RemoteAddr)

	if shouldTrustProxyHeaders(c, directIP) {
		if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
			clientIP := strings.TrimSpace(strings.Split(ip, ",")[0])
			if isValidIP(clientIP) {
				return clientIP
			}
		}
		if ip := c.GetHeader("X-Real-IP"); ip != "" {
			if isValidIP(ip) {
				return ip
			}
		}
	}

	return directIP
}

func shouldTrustProxyHeaders(c *Context, directIP string) bool {
	if trustedProxies, exists := c.Get("trusted_proxies"); exists {
		if proxies, ok := trustedProxies.([]string); ok {
			for _, proxy := range proxies {
				if proxy == directIP {
					return true
				}
				if strings.Contains(proxy, "/") {
					if _, cidr, err := net.ParseCIDR(proxy); err == nil {
						if ip := net.ParseIP(directIP); ip != nil && cidr.Contains(ip) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func (c *Context) BaseURL() string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}

func (c *Context) BodyRaw() ([]byte, error) {
	return io.ReadAll(c.Request.Body)
}

// Optimization: Cache struct field metadata to avoid repeated reflection overhead
var queryDecoderCache sync.Map // map[reflect.Type][]fieldInfo

type fieldInfo struct {
	Index int
	Tag   string
}

func getCachedStructInfo(typ reflect.Type) []fieldInfo {
	if cached, ok := queryDecoderCache.Load(typ); ok {
		return cached.([]fieldInfo)
	}

	var infos []fieldInfo
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("query")
		if tag != "" {
			infos = append(infos, fieldInfo{Index: i, Tag: tag})
		}
	}

	// Store even if empty to avoid re-scanning
	queryDecoderCache.Store(typ, infos)
	return infos
}

func (c *Context) QueryParser(out interface{}) error {
	if err := c.Request.ParseForm(); err != nil {
		return err
	}

	val := reflect.ValueOf(out)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errors.New("QueryParser requires a pointer to a struct")
	}

	elem := val.Elem()
	typ := elem.Type()

	// Optimization: Use cached field info
	fields := getCachedStructInfo(typ)

	for _, info := range fields {
		paramValue := c.Query(info.Tag)
		if paramValue == "" {
			continue
		}

		fieldValue := elem.Field(info.Index)
		if fieldValue.CanSet() {
			switch fieldValue.Kind() {
			case reflect.String:
				fieldValue.SetString(paramValue)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if intVal, err := strconv.ParseInt(paramValue, 10, 64); err == nil {
					fieldValue.SetInt(intVal)
				}
			case reflect.Bool:
				if boolVal, err := strconv.ParseBool(paramValue); err == nil {
					fieldValue.SetBool(boolVal)
				}
			}
		}
	}
	return nil
}

func (c *Context) Hostname() string {
	return c.Request.Host
}

func (c *Context) Is(extension string) bool {
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		return false
	}

	extension = strings.TrimPrefix(extension, ".")

	mimeType := mime.TypeByExtension("." + extension)
	if mimeType == "" {
		// if no MIME type is found, assume the extension is a full MIME
		mimeType = extension
	}

	return strings.HasPrefix(contentType, mimeType)
}

func (c *Context) Protocol() string {
	if c.Request.TLS != nil {
		return "https"
	}
	return "http"
}

func (c *Context) BindJSON(i interface{}) error {
	contentType := c.GetHeader("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return http.ErrNotSupported
	}

	// SECUCHECK: Limit JSON payload size to prevent DoS attacks (default 1MB)
	const maxJSONSize = 1 << 20 // 1MB
	// We read the body into a limit reader, but UnmarshalRead takes a reader directly.
	// Note: UnmarshalRead in v2 consumes the whole reader by default logic or we might need to check.
	// Actually, v2 UnmarshalRead reads until EOF or end of value.

	limitedReader := io.LimitReader(c.Request.Body, maxJSONSize)

	// Optimization: standard library friendly
	decoder := json.NewDecoder(limitedReader)
	// Default validation behavior
	decoder.DisallowUnknownFields()

	return decoder.Decode(i)
}

// BodyParser binds the request body to a struct based on the Content-Type header.
// It supports:
// - application/json -> BindJSON
// - application/x-www-form-urlencoded -> QueryParser (form data)
// - multipart/form-data -> QueryParser (form data)
// - defaults to QueryParser for other types if methods is GET/DELETE
func (c *Context) BodyParser(out interface{}) error {
	ctype := c.GetHeader("Content-Type")

	// JSON
	if strings.HasPrefix(ctype, "application/json") {
		return c.BindJSON(out)
	}

	// Form Data
	if strings.HasPrefix(ctype, "application/x-www-form-urlencoded") || strings.HasPrefix(ctype, "multipart/form-data") {
		// Use QueryParser logic but for form data (which QueryParser internally handles via ParseForm)
		return c.QueryParser(out)
	}

	// Fallback/Default behavior
	// If it's a GET/DELETE request without specific content type, try parsing query params
	if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodDelete {
		if err := c.QueryParser(out); err != nil {
			return err
		}
	} else if strings.HasPrefix(ctype, "application/json") {
		// Already handled above, but for flow completeness
	} else if strings.HasPrefix(ctype, "application/x-www-form-urlencoded") || strings.HasPrefix(ctype, "multipart/form-data") {
		// Already handled
	} else {
		return fmt.Errorf("BodyParser: unsupported content-type: %s", ctype)
	}

	// Validation hook
	if validator, ok := out.(Validator); ok {
		return validator.Validate()
	}

	return nil
}

// Validator is an interface for structs that can validate themselves
type Validator interface {
	Validate() error
}
