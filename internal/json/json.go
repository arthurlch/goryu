package json

import (
	"encoding/json"
	"io"

	"github.com/bytedance/sonic"
)

// Engine represents the JSON encoding/decoding engine
type Engine interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
	NewEncoder(w io.Writer) Encoder
	NewDecoder(r io.Reader) Decoder
}

// Encoder interface for JSON encoding
type Encoder interface {
	Encode(v interface{}) error
}

// Decoder interface for JSON decoding
type Decoder interface {
	Decode(v interface{}) error
}

// defaultEngine uses standard library JSON
type defaultEngine struct{}

func (e defaultEngine) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (e defaultEngine) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (e defaultEngine) NewEncoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}

func (e defaultEngine) NewDecoder(r io.Reader) Decoder {
	return json.NewDecoder(r)
}

// sonicEngine uses bytedance/sonic for high-performance JSON
type sonicEngine struct{}

func (e sonicEngine) Marshal(v interface{}) ([]byte, error) {
	return sonic.Marshal(v)
}

func (e sonicEngine) Unmarshal(data []byte, v interface{}) error {
	return sonic.Unmarshal(data, v)
}

func (e sonicEngine) NewEncoder(w io.Writer) Encoder {
	return sonic.ConfigDefault.NewEncoder(w)
}

func (e sonicEngine) NewDecoder(r io.Reader) Decoder {
	return sonic.ConfigDefault.NewDecoder(r)
}

// Default engine instance - use standard library for now
var Default Engine = defaultEngine{}

// SetDefault sets the default JSON engine
func SetDefault(engine Engine) {
	Default = engine
}

// UseStandardJSON switches to standard library JSON (default)
func UseStandardJSON() {
	Default = defaultEngine{}
}

// UseSonicJSON switches to sonic JSON for potentially better performance on large payloads
func UseSonicJSON() {
	Default = sonicEngine{}
}
