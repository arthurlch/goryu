package bench

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurlch/goryu"
	"github.com/gin-gonic/gin"
)

// BenchmarkGoryu_JSON benchmarks simple JSON response
// I base myself on Gin since it's a popular framework and has good performance
func BenchmarkGoryu_JSON(b *testing.B) {
	// Setup Goryu
	falsePtr := false
	app := goryu.New(goryu.Config{
		EnableMonitoring: &falsePtr,
	})
	app.GET("/hello", func(c *goryu.Context) {
		c.JSON(http.StatusOK, map[string]string{"message": "hello world"})
	})

	req := httptest.NewRequest("GET", "/hello", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Reset recorder for fairness (though overhead applies to both)
		// w = httptest.NewRecorder() // allocating new recorder adds overhead to benchmark, 
		// but reusing it might be unfair if frameworks accumulate data.
		// Standard practice is to reuse or reset.
		// For pure router/framework bench, we want to minimize external allocs.
		// goryu.App.ServeHTTP wraps the whole cycle.
		
		// To be fair and realistic
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

// BenchmarkGin_JSON benchmarks simple JSON response
func BenchmarkGin_JSON(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/hello", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hello world"})
	})

	req := httptest.NewRequest("GET", "/hello", nil)
	
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

type HeavyStruct struct {
	Name    string                 `json:"name"`
	Age     int                    `json:"age"`
	Tags    []string               `json:"tags"`
	Meta    map[string]interface{} `json:"meta"`
	Active  bool                   `json:"active"`
	History []int                  `json:"history"`
}

func getHeavyPayload() HeavyStruct {
	return HeavyStruct{
		Name:    "benchmark-user",
		Age:     30,
		Tags:    []string{"admin", "user", "verified", "premium"},
		Meta:    map[string]interface{}{"ip": "127.0.0.1", "ua": "benchmark-agent"},
		Active:  true,
		History: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
	}
}

// BenchmarkGoryu_JSON_Heavy benchmarks complex JSON response
func BenchmarkGoryu_JSON_Heavy(b *testing.B) {
	// Use Sonic for maximum performance on heavy payloads
	goryu.UseSonicJSON()
	
	falsePtr := false
	app := goryu.New(goryu.Config{
		EnableMonitoring: &falsePtr,
	})
	
	payload := getHeavyPayload()
	
	app.GET("/heavy", func(c *goryu.Context) {
		c.JSON(http.StatusOK, payload)
	})

	req := httptest.NewRequest("GET", "/heavy", nil)
	
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

// BenchmarkGin_JSON_Heavy benchmarks complex JSON response
func BenchmarkGin_JSON_Heavy(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	
	payload := getHeavyPayload()
	
	r.GET("/heavy", func(c *gin.Context) {
		c.JSON(http.StatusOK, payload)
	})

	req := httptest.NewRequest("GET", "/heavy", nil)
	
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkGoryu_Param benchmarks path parameter routing
func BenchmarkGoryu_Param(b *testing.B) {
	falsePtr := false
	app := goryu.New(goryu.Config{
		EnableMonitoring: &falsePtr,
	})
	app.GET("/user/:name", func(c *goryu.Context) {
		name := c.Param("name")
		c.Text(http.StatusOK, name)
	})

	req := httptest.NewRequest("GET", "/user/gordon", nil)
	
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

// BenchmarkGin_Param benchmarks path parameter routing
func BenchmarkGin_Param(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/user/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.String(http.StatusOK, name)
	})

	req := httptest.NewRequest("GET", "/user/gordon", nil)
	
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}
