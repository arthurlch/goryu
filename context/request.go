package context

import (
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"strings"
)

func (c *Context) Query(name string) string {
	return c.Request.URL.Query().Get(name)
}

func (c *Context) Form(name string) string {
	return c.Request.FormValue(name)
}

func (c *Context) Bind(i interface{}) error {
	if c.Request.Header.Get("Content-Type") == "application/json" {
		decoder := json.NewDecoder(c.Request.Body)
		return decoder.Decode(i)
	}
	return http.ErrNotSupported
}

func (c *Context) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return c.Request.FormFile(key)
}

func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err := src.Close(); err != nil {
			log.Printf("Failed to close source file: %v", err)
		}
	}()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Printf("Failed to close destination file: %v", err)
		}
	}()

	_, err = io.Copy(out, src)
	return err
}

func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

func (c *Context) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

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
