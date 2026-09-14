package goryu_test

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurlch/goryu"
	goryuctx "github.com/arthurlch/goryu/goryuctx"
)

type signupReq struct {
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func (s signupReq) Validate() error {
	if s.Email == "" {
		return errors.New("email required")
	}
	return nil
}

func bindContext(body string) *goryu.Ctx {
	req := httptest.NewRequest("POST", "/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return goryuctx.NewContext(httptest.NewRecorder(), req)
}

func TestBindTypedValid(t *testing.T) {
	v, err := goryu.Bind[signupReq](bindContext(`{"email":"a@b.com","age":30}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Email != "a@b.com" || v.Age != 30 {
		t.Fatalf("bound value wrong: %+v", v)
	}
}

func TestBindTypedValidationFails(t *testing.T) {
	_, err := goryu.Bind[signupReq](bindContext(`{"age":30}`))
	if err == nil {
		t.Fatal("expected validation error for missing email")
	}
}
