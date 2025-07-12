package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Agent struct {
	client         *http.Client
	request        *http.Request
	errors         []error
	debugWriter    io.Writer
	jsonEncoder    func(v interface{}) ([]byte, error)
	jsonDecoder    func(data []byte, v interface{}) error
	tlsConfig      *tls.Config
}

func New() *Agent {
	return &Agent{
		client:      &http.Client{},
		errors:      make([]error, 0),
		debugWriter: os.Stdout,
		jsonEncoder: json.Marshal,
		jsonDecoder: json.Unmarshal,
	}
}

func Get(url string) *Agent {
	return New().Get(url)
}

func Post(url string) *Agent {
	return New().Post(url)
}

func Put(url string) *Agent {
	return New().Put(url)
}

func Delete(url string) *Agent {
	return New().Delete(url)
}

func Patch(url string) *Agent {
	return New().Patch(url)
}

func Head(url string) *Agent {
	return New().Head(url)
}

// agents methods 

func (a *Agent) Get(url string) *Agent {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		a.errors = append(a.errors, err)
	}
	a.request = req
	return a
}

func (a *Agent) Post(url string) *Agent {
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		a.errors = append(a.errors, err)
	}
	a.request = req
	return a
}

func (a *Agent) Put(url string) *Agent {
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		a.errors = append(a.errors, err)
	}
	a.request = req
	return a
}

func (a *Agent) Delete(url string) *Agent {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		a.errors = append(a.errors, err)
	}
	a.request = req
	return a
}

func (a *Agent) Patch(url string) *Agent {
	req, err := http.NewRequest(http.MethodPatch, url, nil)
	if err != nil {
		a.errors = append(a.errors, err)
	}
	a.request = req
	return a
}

func (a *Agent) Head(url string) *Agent {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		a.errors = append(a.errors, err)
	}
	a.request = req
	return a
}

func (a *Agent) Set(key, value string) *Agent {
	a.request.Header.Set(key, value)
	return a
}

func (a *Agent) Add(key, value string) *Agent {
	a.request.Header.Add(key, value)
	return a
}

func (a *Agent) Body(body []byte) *Agent {
	a.request.Body = io.NopCloser(bytes.NewReader(body))
	a.request.ContentLength = int64(len(body))
	return a
}

func (a *Agent) JSON(v interface{}) *Agent {
	b, err := a.jsonEncoder(v)
	if err != nil {
		a.errors = append(a.errors, err)
		return a
	}
	a.Set("Content-Type", "application/json")
	return a.Body(b)
}

func (a *Agent) String() (int, string, []error) {
	code, body, errs := a.Bytes()
	return code, string(body), errs
}

func (a *Agent) Bytes() (int, []byte, []error) {
	if len(a.errors) > 0 {
		return 0, nil, a.errors
	}

	resp, err := a.client.Do(a.request)
	if err != nil {
		a.errors = append(a.errors, err)
		return 0, nil, a.errors
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		a.errors = append(a.errors, err)
		return resp.StatusCode, nil, a.errors
	}

	return resp.StatusCode, body, nil
}

func (a *Agent) Struct(v interface{}) (int, []byte, []error) {
	code, body, errs := a.Bytes()
	if len(errs) > 0 {
		return code, body, errs
	}

	if err := a.jsonDecoder(body, v); err != nil {
		a.errors = append(a.errors, err)
	}

	return code, body, a.errors
}

func (a *Agent) UserAgent(userAgent string) *Agent {
	return a.Set("User-Agent", userAgent)
}

func (a *Agent) ContentType(contentType string) *Agent {
	return a.Set("Content-Type", contentType)
}

func (a *Agent) Cookie(key, value string) *Agent {
	a.request.AddCookie(&http.Cookie{Name: key, Value: value})
	return a
}

func (a *Agent) BasicAuth(username, password string) *Agent {
	a.request.SetBasicAuth(username, password)
	return a
}

func (a *Agent) Query(key, value string) *Agent {
	params := a.request.URL.Query()
	params.Add(key, value)
	a.request.URL.RawQuery = params.Encode()
	return a
}

func (a *Agent) Form(data url.Values) *Agent {
	a.ContentType("application/x-www-form-urlencoded")
	return a.Body([]byte(data.Encode()))
}

// Timeout sets the timeout for the request.
func (a *Agent) Timeout(timeout time.Duration) *Agent {
	a.client.Timeout = timeout
	return a
}

func (a *Agent) InsecureSkipVerify() *Agent {
	if a.tlsConfig == nil {
		a.tlsConfig = &tls.Config{}
	}
	a.tlsConfig.InsecureSkipVerify = true
	if transport, ok := a.client.Transport.(*http.Transport); ok {
		transport.TLSClientConfig = a.tlsConfig
	} else {
		a.client.Transport = &http.Transport{
			TLSClientConfig: a.tlsConfig,
		}
	}
	return a
}

// might add more methods need proposal