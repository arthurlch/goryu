package context

import (
	"encoding/json"
	"log"
	"net/http"
)

func (c *Context) JSON(code int, obj interface{}) {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(code)
	if err := json.NewEncoder(c.Writer).Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
	}
}

func (c *Context) Text(code int, text string) {
	c.Writer.Header().Set("Content-Type", "text/plain")
	c.Writer.WriteHeader(code)
	if _, err := c.Writer.Write([]byte(text)); err != nil {
		log.Printf("Error writing text response: %v", err)
	}
}

func (c *Context) Data(code int, contentType string, data []byte) {
	c.Writer.Header().Set("Content-Type", contentType)
	c.Writer.WriteHeader(code)
	if _, err := c.Writer.Write(data); err != nil {
		log.Printf("Error writing data response: %v", err)
	}
}

func (c *Context) Status(code int) {
	c.Writer.WriteHeader(code)
}

func (c *Context) Redirect(code int, location string) {
	http.Redirect(c.Writer, c.Request, location, code)
}

func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Writer, cookie)
}

func (c *Context) Error(err error) {
	log.Println("Error:", err)
	http.Error(c.Writer, "Internal Server Error", http.StatusInternalServerError)
}

func (c *Context) AbortWithStatus(code int) {
	c.Writer.WriteHeader(code)
}

func (c *Context) AbortWithStatusJSON(code int, data interface{}) {
	c.JSON(code, data)
}
