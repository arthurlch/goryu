package context

import (
	"encoding/json"
	"errors"
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

// preventing path traversal attacks.
// It sanitizes the filename to prevent path traversal attacks.
func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dstFilename string) error {
	const uploadDir = "uploads"

	if strings.Contains(dstFilename, "/") || strings.Contains(dstFilename, "\\") {
		return errors.New("invalid destination filename: contains path separators")
	}

	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return err
	}

	safePath := filepath.Join(uploadDir, dstFilename)

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	out, err := os.Create(safePath)
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

func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

func (c *Context) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

// RemoteIP returns the client's IP address.
// IMPORTANT: This function trusts HTTP headers like X-Forwarded-For and X-Real-IP.
// It should only be used when the server is behind a trusted reverse proxy that
// sets these headers correctly. Otherwise, the IP can be easily spoofed =X
func (c *Context) RemoteIP() string {
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}
	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	return ip
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

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("query")
		if tag == "" {
			continue
		}

		paramValue := c.Query(tag)
		if paramValue == "" {
			continue
		}

		fieldValue := elem.Field(i)
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
		// If no MIME type is found, assume the extension is a full MIME
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

	decoder := json.NewDecoder(c.Request.Body)
	// For security, you might want to disallow unknown fields to prevent
	// unexpected data from being processed !
	// decoder.DisallowUnknownFields()
	return decoder.Decode(i)
}
