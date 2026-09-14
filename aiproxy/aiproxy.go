// Package aiproxy is a small streaming reverse-proxy helper for LLM/AI
// providers. It forwards the current request to an upstream provider and streams
// the response back to the client, flushing as data arrives — so SSE and
// chunked token streams pass straight through.
//
//	px := aiproxy.New(aiproxy.Config{
//	    BaseURL: "https://api.openai.com",
//	    APIKey:  os.Getenv("OPENAI_API_KEY"),
//	})
//	app.POST("/v1/chat/completions", func(c *goryu.Ctx) {
//	    _ = px.Stream(c, "/v1/chat/completions")
//	})
package aiproxy

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
)

type Config struct {
	// BaseURL is the upstream provider origin, e.g. https://api.openai.com.
	BaseURL string
	// APIKey, when set, is attached as "<AuthScheme> <APIKey>" on AuthHeader.
	APIKey string
	// AuthHeader defaults to "Authorization".
	AuthHeader string
	// AuthScheme defaults to "Bearer".
	AuthScheme string
	// Header holds extra headers added to every upstream request.
	Header http.Header
	// Client is the HTTP client used upstream. Defaults to one with Timeout.
	Client *http.Client
	// Timeout for non-streaming Forward calls. Streaming uses no client timeout
	// (it relies on the request context). Default 60s.
	Timeout time.Duration
}

type Proxy struct {
	baseURL    string
	apiKey     string
	authHeader string
	authScheme string
	header     http.Header
	stream     *http.Client
	forward    *http.Client
}

var hopHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Te":                  true,
	"Trailer":             true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
}

// clientSecretHeaders are stripped from the client request before it is
// forwarded upstream, so a caller's own credentials and session are never leaked
// to the third-party provider. The proxy adds its own credential via AuthHeader.
var clientSecretHeaders = map[string]bool{
	"Authorization": true,
	"Cookie":        true,
	"X-Api-Key":     true,
}

func New(cfg Config) *Proxy {
	if cfg.AuthHeader == "" {
		cfg.AuthHeader = "Authorization"
	}
	if cfg.AuthScheme == "" {
		cfg.AuthScheme = "Bearer"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}
	streamClient := cfg.Client
	if streamClient == nil {
		streamClient = &http.Client{}
	}
	forwardClient := cfg.Client
	if forwardClient == nil {
		forwardClient = &http.Client{Timeout: cfg.Timeout}
	}
	return &Proxy{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		authHeader: cfg.AuthHeader,
		authScheme: cfg.AuthScheme,
		header:     cfg.Header,
		stream:     streamClient,
		forward:    forwardClient,
	}
}

func (p *Proxy) Stream(c *context.Context, path string) error {
	resp, err := p.do(c, path, p.stream)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	copyResponseHeaders(c, resp)
	c.Writer.WriteHeader(resp.StatusCode)

	flusher, _ := c.Writer.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		if ctxErr := c.Request.Context().Err(); ctxErr != nil {
			return ctxErr
		}
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := c.Writer.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func (p *Proxy) Forward(c *context.Context, path string) error {
	resp, err := p.do(c, path, p.forward)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	copyResponseHeaders(c, resp)
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = io.Copy(c.Writer, resp.Body)
	return err
}

func (p *Proxy) do(c *context.Context, path string, client *http.Client) (*http.Response, error) {
	if p.baseURL == "" {
		return nil, errors.New("aiproxy: BaseURL is empty")
	}
	url := p.baseURL + "/" + strings.TrimLeft(path, "/")

	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, url, c.Request.Body)
	if err != nil {
		return nil, err
	}
	for k, v := range c.Request.Header {
		canonical := http.CanonicalHeaderKey(k)
		if hopHeaders[canonical] || clientSecretHeaders[canonical] {
			continue
		}
		req.Header[k] = append([]string(nil), v...)
	}
	for k, v := range p.header {
		req.Header[k] = append([]string(nil), v...)
	}
	if p.apiKey != "" {
		req.Header.Set(p.authHeader, p.authScheme+" "+p.apiKey)
	}
	return client.Do(req)
}

func copyResponseHeaders(c *context.Context, resp *http.Response) {
	dst := c.Writer.Header()
	for k, v := range resp.Header {
		if hopHeaders[http.CanonicalHeaderKey(k)] {
			continue
		}
		dst[k] = append([]string(nil), v...)
	}
}
