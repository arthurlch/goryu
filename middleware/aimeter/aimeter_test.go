package aimeter_test

import (
	"net/http/httptest"
	"testing"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/aimeter"
)

func newCtx(apiKey string) (*context.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/chat", nil)
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	return context.NewContext(w, req), w
}

func TestMeterHeadersAndCost(t *testing.T) {
	var got aimeter.Result
	mw := aimeter.New(aimeter.Config{
		PromptCostPer1K:     0.01,
		CompletionCostPer1K: 0.03,
		OnResult:            func(c *context.Context, r aimeter.Result) { got = r },
	})
	handler := mw(func(c *context.Context) {
		aimeter.SetUsage(c, aimeter.Usage{PromptTokens: 1000, CompletionTokens: 1000})
	})

	c, w := newCtx("k1")
	handler(c)

	if got.Usage.Total() != 2000 {
		t.Fatalf("expected 2000 tokens, got %d", got.Usage.Total())
	}
	// 1000/1000*0.01 + 1000/1000*0.03 = 0.04
	if got.CostUSD < 0.0399 || got.CostUSD > 0.0401 {
		t.Fatalf("cost wrong: %v", got.CostUSD)
	}
	if w.Header().Get("X-Tokens-Total") != "2000" {
		t.Fatalf("header wrong: %q", w.Header().Get("X-Tokens-Total"))
	}
}

func TestMeterRequestLimit(t *testing.T) {
	mw := aimeter.New(aimeter.Config{MaxRequests: 2, Window: time.Minute})
	handler := mw(func(c *context.Context) { c.Text(200, "ok") })

	for i := range 2 {
		c, w := newCtx("same")
		handler(c)
		if w.Code != 200 {
			t.Fatalf("request %d should pass, got %d", i, w.Code)
		}
	}
	c, w := newCtx("same")
	handler(c)
	if w.Code != 429 {
		t.Fatalf("3rd request should be limited, got %d", w.Code)
	}
}

func TestMeterTokenBudget(t *testing.T) {
	mw := aimeter.New(aimeter.Config{MaxTokens: 1500, Window: time.Minute})
	handler := mw(func(c *context.Context) {
		aimeter.SetUsage(c, aimeter.Usage{PromptTokens: 1000, CompletionTokens: 0})
	})

	// First request consumes 1000 tokens (allowed).
	c1, w1 := newCtx("kb")
	handler(c1)
	if w1.Code == 429 {
		t.Fatal("first request should not be limited")
	}
	// Second request: budget still under 1500, allowed, pushes total to 2000.
	c2, _ := newCtx("kb")
	handler(c2)
	// Third request: budget (2000) exceeds 1500 -> rejected.
	c3, w3 := newCtx("kb")
	handler(c3)
	if w3.Code != 429 {
		t.Fatalf("request should be rejected once token budget exceeded, got %d", w3.Code)
	}
}

func TestMeterPerKeyIsolation(t *testing.T) {
	mw := aimeter.New(aimeter.Config{MaxRequests: 1, Window: time.Minute})
	handler := mw(func(c *context.Context) { c.Text(200, "ok") })

	c1, w1 := newCtx("a")
	handler(c1)
	c2, w2 := newCtx("b")
	handler(c2)
	if w1.Code != 200 || w2.Code != 200 {
		t.Fatalf("different keys must not share limits: %d %d", w1.Code, w2.Code)
	}
}
