# Goryu Error Handling Middleware

An elegant error handling middleware that makes working with errors in Go less verbose and more structured.

## Features

- 🎯 **Structured Error Types** - Rich error objects with codes, messages, and metadata
- 🔧 **Fluent API** - Chain error building with a clean, readable syntax
- 🎨 **Multiple Usage Patterns** - Traditional, functional, or fluent styles
- 🛡️ **Panic Recovery** - Automatic panic handling with stack traces
- 📝 **Detailed Logging** - Configurable error logging with request tracking
- 🔍 **Development Mode** - Extra debugging information when needed
- ⚡ **Zero Allocation** - Efficient error handling without performance impact

## Installation

```go
import "github.com/arthurlch/goryu/middleware/errors"
```

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/middleware/errors"
)

func main() {
    app := goryu.New()
    
    // Add error handling middleware
    app.Use(errors.New())
    
    // Traditional error return pattern
    app.GET("/users/:id", errors.Handle(func(c *goryu.Context) error {
        id := c.Param("id")
        
        user, err := getUserByID(id)
        if err != nil {
            return errors.NotFound("user")
        }
        
        return c.JSON(200, user)
    }))
    
    app.Run(":8080")
}
```

### Fluent Error API

```go
app.POST("/login", func(c *goryu.Context) {
    var req LoginRequest
    
    // Validation errors
    if err := c.Bind(&req); err != nil {
        errors.Error(c).BadRequest("Invalid request body")
        return
    }
    
    if req.Email == "" {
        errors.Error(c).Validation("email", "Email is required")
        return
    }
    
    // Business logic errors
    user, err := authenticateUser(req.Email, req.Password)
    if err != nil {
        errors.Error(c).Unauthorized("Invalid credentials")
        return
    }
    
    c.JSON(200, user)
})
```

## Configuration

```go
app.Use(errors.NewWithConfig(errors.Config{
    ShowDetails:    true,  // Show error details in response
    ShowStackTrace: false, // Include stack traces (only in dev mode)
    LogErrors:      true,  // Log errors to console
    DevMode:        false, // Development mode with extra debugging
    
    // Custom error handler
    CustomHandler: func(c *goryu.Context, err error) {
        // Custom error handling logic
        logger.Error("Request failed", "error", err)
    },
    
    // Transform errors before sending response
    ErrorTransformer: func(err error) error {
        // Transform or wrap errors
        return err
    },
    
    // Custom response format
    ResponseFormatter: func(c *goryu.Context, err *errors.AppError) interface{} {
        return map[string]interface{}{
            "success": false,
            "error":   err,
        }
    },
}))
```

## Error Types

### Pre-built Error Constructors

```go
// HTTP 400 Bad Request
errors.BadRequest("Invalid input format")

// HTTP 401 Unauthorized
errors.Unauthorized("Please authenticate")

// HTTP 403 Forbidden
errors.Forbidden("Insufficient permissions")

// HTTP 404 Not Found
errors.NotFound("user") // "user not found"

// HTTP 409 Conflict
errors.Conflict("Email already exists")

// HTTP 400 with validation details
errors.ValidationError("email", "Invalid email format")

// HTTP 500 Internal Server Error
errors.InternalError(err) // Wraps internal errors
```

### Custom Errors with Builder

```go
err := errors.NewError("PAYMENT_FAILED", "Payment processing failed").
    Status(http.StatusPaymentRequired).
    Detail("amount", 99.99).
    Detail("currency", "USD").
    Detail("processor", "stripe").
    Internal(stripeErr).
    RequestID(c.Get("request_id").(string)).
    Source("payment_service.go:123").
    Build()
```

## Error Handling Patterns

### 1. Traditional Pattern with Error Returns

```go
app.GET("/orders/:id", errors.Handle(func(c *goryu.Context) error {
    orderID := c.Param("id")
    
    order, err := db.GetOrder(orderID)
    if err != nil {
        if err == sql.ErrNoRows {
            return errors.NotFound("order")
        }
        return errors.InternalError(err)
    }
    
    // Check permissions
    if !canViewOrder(c, order) {
        return errors.Forbidden("Cannot view this order")
    }
    
    return c.JSON(200, order)
}))
```

### 2. Fluent API Pattern

```go
app.PUT("/profile", func(c *goryu.Context) {
    var profile ProfileUpdate
    
    // Quick error responses
    if err := c.Bind(&profile); err != nil {
        errors.Error(c).BadRequest("Invalid profile data")
        return
    }
    
    if err := validateProfile(profile); err != nil {
        errors.Error(c).Validation("profile", err.Error())
        return
    }
    
    if err := updateProfile(c.UserID(), profile); err != nil {
        errors.Error(c).Internal(err)
        return
    }
    
    c.JSON(200, map[string]string{"status": "updated"})
})
```

### 3. Chain Pattern for Complex Operations

```go
app.POST("/checkout", func(c *goryu.Context) {
    var order CheckoutRequest
    
    chain := errors.NewChain(c).
        Do(func() error {
            return c.Bind(&order)
        }).
        Do(func() error {
            return validateOrder(order)
        }).
        DoWithResult(func() (interface{}, error) {
            return processPayment(order)
        }).
        Do(func() error {
            result, _ := chain.Result()
            payment := result.(*PaymentResult)
            return createOrder(order, payment)
        }).
        OnSuccess(func() {
            c.JSON(200, map[string]string{
                "status": "success",
                "order_id": order.ID,
            })
        })
    
    // Sends error if any operation failed
    chain.SendError("CHECKOUT_FAILED", "Failed to process checkout")
})
```

### 4. Helper Functions for Common Patterns

```go
app.GET("/report", func(c *goryu.Context) {
    // Handle result/error pairs elegantly
    errors.HandleResult(c, generateReport(), func(report *Report) {
        c.JSON(200, report)
    })
})

app.POST("/data", func(c *goryu.Context) {
    var data DataInput
    
    // Validate and handle in one step
    if !errors.ValidateAndHandle(c, func() error {
        if err := c.Bind(&data); err != nil {
            return err
        }
        return data.Validate()
    }) {
        return // Error already sent
    }
    
    // Process valid data...
})
```

## Working with Error Details

```go
// Create detailed errors for better debugging
err := errors.NewError("INTEGRATION_ERROR", "External service failed").
    Status(http.StatusBadGateway).
    Details(map[string]interface{}{
        "service": "payment-gateway",
        "endpoint": "https://api.payment.com/charge",
        "timeout": "30s",
        "retry_count": 3,
    }).
    Internal(originalError).
    Build()

// Errors are automatically enriched with:
// - Timestamp
// - Request ID (if using request ID middleware)
// - Stack trace (in dev mode)
// - Source location
```

## Panic Recovery

The middleware automatically handles panics:

```go
app.GET("/dangerous", func(c *goryu.Context) {
    // This panic will be caught and converted to a 500 error
    panic("something went wrong")
})

// Response:
{
    "error": {
        "code": "PANIC",
        "message": "Internal server error",
        "request_id": "req-123",
        "timestamp": "2024-01-20T10:30:00Z"
    }
}
```

## Multiple Errors

Handle multiple errors in a single response:

```go
app.POST("/bulk-upload", func(c *goryu.Context) {
    files := c.MultipartForm().Files["files"]
    
    for i, file := range files {
        if err := processFile(file); err != nil {
            c.Error(errors.ValidationError(
                fmt.Sprintf("file[%d]", i),
                err.Error(),
            ))
        }
    }
    
    if c.HasErrors() {
        // Middleware will send all collected errors
        return
    }
    
    c.JSON(200, map[string]int{
        "processed": len(files),
    })
})
```

## Best Practices

### 1. Use Semantic Error Codes

```go
// Good - specific and searchable
errors.NewError("USER_EMAIL_EXISTS", "This email is already registered")
errors.NewError("PAYMENT_INSUFFICIENT_FUNDS", "Insufficient funds")

// Avoid - too generic
errors.NewError("ERROR", "Something went wrong")
errors.NewError("FAIL", "Operation failed")
```

### 2. Provide Helpful Details

```go
// Include relevant context
errors.NotFound("user").
    WithDetail("user_id", userID).
    WithDetail("search_method", "email")
```

### 3. Use Appropriate HTTP Status Codes

```go
// User errors: 4xx
errors.BadRequest()      // 400 - Client sent bad data
errors.Unauthorized()    // 401 - Not authenticated
errors.Forbidden()       // 403 - Not authorized
errors.NotFound()        // 404 - Resource not found
errors.Conflict()        // 409 - Conflict with current state

// Server errors: 5xx
errors.InternalError()   // 500 - Server error
errors.NewError().Status(502) // 502 - Bad Gateway
errors.NewError().Status(503) // 503 - Service Unavailable
```

### 4. Don't Expose Internal Errors

```go
// In production (DevMode: false)
user, err := db.Query("SELECT * FROM users WHERE id = ?", id)
if err != nil {
    // Internal error is logged but not sent to client
    return errors.InternalError(err)
}

// Response shows generic message:
{
    "error": {
        "code": "INTERNAL_ERROR",
        "message": "An internal error occurred"
    }
}

// In development (DevMode: true), internal details are included
```

## Integration with Other Middleware

### With Request ID Middleware

```go
app.Use(requestid.New())
app.Use(errors.New())

// Errors automatically include request ID
```

### With Logger Middleware

```go
app.Use(logger.New())
app.Use(errors.NewWithConfig(errors.Config{
    CustomHandler: func(c *goryu.Context, err error) {
        c.Logger().Error("Request failed",
            "error", err,
            "path", c.Path(),
            "method", c.Method(),
        )
    },
}))
```

### With Metrics Middleware

```go
app.Use(metrics.New())
app.Use(errors.NewWithConfig(errors.Config{
    CustomHandler: func(c *goryu.Context, err error) {
        // Track error metrics
        errorCounter.WithLabelValues(err.(*errors.AppError).Code).Inc()
    },
}))
```

## Testing

Easy to test with the error middleware:

```go
func TestUserEndpoint(t *testing.T) {
    app := goryu.New()
    app.Use(errors.New())
    
    app.GET("/users/:id", errors.Handle(func(c *goryu.Context) error {
        id := c.Param("id")
        if id == "999" {
            return errors.NotFound("user")
        }
        return c.JSON(200, map[string]string{"id": id})
    }))
    
    // Test not found
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/users/999", nil)
    app.ServeHTTP(w, req)
    
    assert.Equal(t, 404, w.Code)
    
    var response map[string]interface{}
    json.NewDecoder(w.Body).Decode(&response)
    
    assert.Equal(t, "NOT_FOUND", response["error"].(map[string]interface{})["code"])
}
```

## Performance Considerations

The error middleware is designed to be lightweight:

- Zero allocations for success paths
- Minimal overhead for error paths
- Efficient panic recovery
- Optional features can be disabled

```go
// Minimal configuration for production
app.Use(errors.NewWithConfig(errors.Config{
    ShowDetails:    false, // Less data in responses
    ShowStackTrace: false, // No stack traces
    LogErrors:      true,  // Still log for monitoring
    DevMode:        false, // Production mode
}))
```

## Examples

See the [examples](examples/) directory for complete applications demonstrating:

- REST API with comprehensive error handling
- GraphQL API with error mapping
- gRPC service with error translation
- WebSocket server with error events
- Microservice with distributed tracing

## License

This middleware is part of the Goryu web framework and follows the same license terms.