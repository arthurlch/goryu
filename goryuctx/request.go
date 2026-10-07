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
	const maxFileSize = 50 << 20

	if err := validateUploadFilename(dstFilename); err != nil {
		return err
	}

	if file.Size > maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d bytes)", file.Size, maxFileSize)
	}

	if len(dstFilename) > maxFilenameLength {
		return errors.New("filename too long")
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return err
	}

	cleanFilename, err := validateAndSanitizeUploadPath(dstFilename)
	if err != nil {
		return fmt.Errorf("invalid filename: %w", err)
	}

	safePath := filepath.Join(uploadDir, cleanFilename)

	absUploadDir, err := filepath.Abs(uploadDir)
	if err != nil {
		return fmt.Errorf("failed to resolve upload directory: %w", err)
	}

	absSafePath, err := filepath.Abs(safePath)
	if err != nil {
		return fmt.Errorf("failed to resolve destination path: %w", err)
	}

	if !strings.HasPrefix(absSafePath, absUploadDir+string(filepath.Separator)) && absSafePath != absUploadDir {
		return errors.New("invalid destination: path traversal detected")
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	out, err := os.OpenFile(absSafePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, src)
	closeErr := out.Close()

	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func validateUploadFilename(filename string) error {
	if filename == "" {
		return errors.New("filename cannot be empty")
	}

	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return errors.New("invalid destination filename: contains path separators")
	}

	if strings.HasPrefix(filename, ".") {
		return errors.New("invalid destination filename: hidden files not allowed")
	}

	dangerousChars := []string{"\x00", "<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range dangerousChars {
		if strings.Contains(filename, char) {
			return fmt.Errorf("invalid destination filename: contains dangerous character '%s'", char)
		}
	}

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

func validateAndSanitizeUploadPath(filename string) (string, error) {
	if filename == "" {
		return "", errors.New("filename cannot be empty")
	}

	if len(filename) > 255 {
		return "", errors.New("filename too long")
	}

	if !utf8.ValidString(filename) {
		return "", errors.New("filename contains invalid UTF-8 characters")
	}

	normalized := norm.NFC.String(filename)

	traversalPatterns := []string{
		"..",
		"%2e%2e",
		"%252e%252e",
		"..%2f",
		"%2e.",
		".%2e",
		"..\\",
		"..%5c",
		"\\u002e\\u002e",
		"..",
		"\u2024",
		"\uFF0E",
	}

	lowerFilename := strings.ToLower(normalized)
	for _, pattern := range traversalPatterns {
		if strings.Contains(lowerFilename, strings.ToLower(pattern)) {
			return "", errors.New("filename contains path traversal patterns")
		}
	}

	suspiciousChars := []string{
		"\x00",
		"\r",
		"\n",
		"\t",
		"<",
		">",
		":",
		"|",
		"?",
		"*",
		"\"",
	}

	for _, char := range suspiciousChars {
		if strings.Contains(normalized, char) {
			return "", fmt.Errorf("filename contains suspicious character: %s", char)
		}
	}

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

	if strings.HasPrefix(normalized, ".") {
		return "", errors.New("hidden files (starting with '.') are not allowed")
	}

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

	cleaned := filepath.Clean(normalized)

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

func (c *Context) RemoteIP() string {
	directIP, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		directIP = c.Request.RemoteAddr
	}

	if shouldTrustProxyHeaders(c, directIP) {
		if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
			if ip := parseIP(strings.Split(forwarded, ",")[0]); ip != "" {
				return ip
			}
		}
		if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
			if ip := parseIP(realIP); ip != "" {
				return ip
			}
		}
	}

	return directIP
}

func shouldTrustProxyHeaders(c *Context, directIP string) bool {
	trustedProxies, exists := c.Get("trusted_proxies")
	if !exists {
		return false
	}

	proxies, ok := trustedProxies.([]string)
	if !ok {
		return false
	}

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
	return false
}

func parseIP(raw string) string {
	if ip := net.ParseIP(strings.TrimSpace(raw)); ip != nil {
		return ip.String()
	}
	return ""
}

func (c *Context) BaseURL() string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}

const maxBodySize = 10 << 20

func (c *Context) BodyRaw() ([]byte, error) {
	return io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxBodySize))
}

var queryDecoderCache sync.Map

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

	queryDecoderCache.Store(typ, infos)
	return infos
}

func (c *Context) QueryParser(out interface{}) error {
	if c.Request.Body != nil {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodySize)
	}
	if err := c.Request.ParseForm(); err != nil {
		return err
	}

	val := reflect.ValueOf(out)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errors.New("QueryParser requires a pointer to a struct")
	}

	elem := val.Elem()
	typ := elem.Type()

	fields := getCachedStructInfo(typ)

	for _, info := range fields {
		paramValue := c.Query(info.Tag)
		if paramValue == "" {
			continue
		}

		fieldValue := elem.Field(info.Index)
		if !fieldValue.CanSet() {
			continue
		}
		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(paramValue)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal, err := strconv.ParseInt(paramValue, 10, fieldValue.Type().Bits())
			if err != nil {
				return fmt.Errorf("invalid value for %q: %w", info.Tag, err)
			}
			fieldValue.SetInt(intVal)
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(paramValue)
			if err != nil {
				return fmt.Errorf("invalid value for %q: %w", info.Tag, err)
			}
			fieldValue.SetBool(boolVal)
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

	const maxJSONSize = 1 << 20
	limitedReader := io.LimitReader(c.Request.Body, maxJSONSize)

	decoder := json.NewDecoder(limitedReader)
	decoder.DisallowUnknownFields()

	return decoder.Decode(i)
}

func (c *Context) BodyParser(out interface{}) error {
	ctype := c.GetHeader("Content-Type")

	switch {
	case strings.HasPrefix(ctype, "application/json"):
		if err := c.BindJSON(out); err != nil {
			return err
		}
	case strings.HasPrefix(ctype, "application/x-www-form-urlencoded"),
		strings.HasPrefix(ctype, "multipart/form-data"):
		if err := c.QueryParser(out); err != nil {
			return err
		}
	case c.Request.Method == http.MethodGet || c.Request.Method == http.MethodDelete:
		if err := c.QueryParser(out); err != nil {
			return err
		}
	default:
		return fmt.Errorf("BodyParser: unsupported content-type: %s", ctype)
	}

	if validator, ok := out.(Validator); ok {
		return validator.Validate()
	}
	return nil
}

type Validator interface {
	Validate() error
}
