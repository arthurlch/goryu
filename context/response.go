package context

import (
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (c *Context) JSON(code int, obj interface{}) error {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(code)
	return json.NewEncoder(c.Writer).Encode(obj)
}

func (c *Context) Text(code int, text string) error {
	c.Writer.Header().Set("Content-Type", "text/plain")
	c.Writer.WriteHeader(code)
	_, err := c.Writer.Write([]byte(text))
	if err != nil {
		log.Printf("Error writing text response: %v", err)
		return err
	}
	return nil
}

func (c *Context) Data(code int, contentType string, data []byte) error {
	c.Writer.Header().Set("Content-Type", contentType)
	c.Writer.WriteHeader(code)
	_, err := c.Writer.Write(data)
	if err != nil {
		log.Printf("Error writing data response: %v", err)
		return err
	}
	return nil
}

func (c *Context) Status(code int) *Context {
	c.Writer.WriteHeader(code)
	return c
}

func (c *Context) Redirect(code int, location string) {
	http.Redirect(c.Writer, c.Request, location, code)
}

func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Writer, cookie)
}

// error is a helper for handling internal server errors.
// error handling in Go. ..
func (c *Context) Error(err error) {
	log.Println("Error:", err)
	http.Error(c.Writer, "Internal Server Error", http.StatusInternalServerError)
}

func (c *Context) ClearCookie(name string) {
	cookie := &http.Cookie{
		Name:    name,
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
		MaxAge:  -1,
	}
	http.SetCookie(c.Writer, cookie)
}

func (c *Context) Attachment(filename ...string) {
	disposition := "attachment"
	if len(filename) > 0 {
		// again using filepath.Base to prevent directory traversal attacks ......
		fname := filepath.Base(filename[0])
		disposition = fmt.Sprintf("attachment; filename=\"%s\"", fname)
	}
	c.Writer.Header().Set("Content-Disposition", disposition)
}

func (c *Context) SendFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		http.Error(c.Writer, "File Not Found", http.StatusNotFound)
		return
	}
	defer func() { _ = file.Close() }()

	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(c.Writer, "Could Not Stat File", http.StatusInternalServerError)
		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(fileInfo.Name()))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Writer.Header().Set("Content-Type", contentType)

	http.ServeContent(c.Writer, c.Request, fileInfo.Name(), fileInfo.ModTime(), file)
}

func (c *Context) Location(path string) {
	c.Writer.Header().Set("Location", path)
}

func (c *Context) Type(ext string) *Context {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	contentType := mime.TypeByExtension(ext)
	if contentType != "" {
		c.Writer.Header().Set("Content-Type", contentType)
	}
	return c
}

func (c *Context) Vary(fields ...string) {
	for _, field := range fields {
		c.Writer.Header().Add("Vary", field)
	}
}

func (c *Context) Append(field string, values ...string) {
	for _, value := range values {
		c.Writer.Header().Add(field, value)
	}
}
