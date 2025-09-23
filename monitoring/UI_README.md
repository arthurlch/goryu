# Goryu Monitoring Dashboard UI

A beautiful, responsive web interface for monitoring your Goryu applications in real-time.

## 🎯 Features

- **Real-time Dashboard** - Auto-refreshing every 5 seconds
- **Responsive Design** - Works on desktop, tablet, and mobile
- **Health Status Overview** - Visual status indicators with color coding
- **Live Metrics** - Memory usage, request counts, error rates, uptime
- **Event Stream** - Real-time application events with filtering
- **Health Check Details** - Individual component health status
- **Modern UI** - Clean, professional design with smooth animations

## 🚀 Quick Start

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/monitoring" 
)

func main() {
    app := goryu.New(goryu.Config{
        AppName: "My Application",
    })
    
    // Enable monitoring with UI
    app.EnableMonitoring("/_monitor")
    
    // Add some health checks
    app.AddHealthCheck("database", &monitoring.HealthCheck{
        Check: func() (monitoring.HealthStatus, error) {
            // Your health check logic here
            return monitoring.StatusHealthy, nil
        },
        Critical: true,
    })
    
    app.Listen(":8080")
}
```

Then visit: **http://localhost:8080/_monitor/ui**

## 📋 Available Endpoints

When you call `app.EnableMonitoring("/_monitor")`, you get:

- `/_monitor` → Redirects to UI dashboard
- `/_monitor/ui` → **Web Dashboard UI**
- `/_monitor/health` → JSON health status
- `/_monitor/metrics` → JSON metrics data
- `/_monitor/events` → JSON events feed
- `/_monitor/dashboard` → JSON overview

## 🎨 Dashboard Sections

### 1. Status Overview Cards
- **Overall Status** - Healthy/Degraded/Unhealthy with color coding
- **Uptime** - How long the application has been running
- **Total Requests** - Number of HTTP requests processed
- **Error Count** - Number of error responses

### 2. Health Checks Panel
- Individual health check status
- Critical vs non-critical indicators
- Response times for each check
- Error messages when checks fail

### 3. Recent Events Panel
- Real-time event stream
- Color-coded event types (requests, errors, custom events)
- Event details and timestamps
- Last 10 events displayed

### 4. System Metrics Panel
- **Memory Usage** - Current memory consumption
- **Goroutines** - Number of active goroutines
- **Average Response Time** - Mean response time
- **Error Rate** - Percentage of requests that failed

## 🎯 Health Status Color Coding

- 🟢 **Green (Healthy)** - All systems operational
- 🟡 **Yellow (Degraded)** - Some issues but still functional
- 🔴 **Red (Unhealthy)** - Critical systems down

## ⚡ Auto-Refresh Features

- **Smart Refresh** - Pauses when tab is hidden, resumes when visible
- **5-Second Intervals** - Real-time updates without overwhelming the server
- **Manual Refresh** - Press 'R' key for instant refresh
- **Loading Indicators** - Visual feedback during refresh

## 📱 Responsive Design

The dashboard adapts to different screen sizes:

- **Desktop** - Full two-column layout with all panels
- **Tablet** - Single column layout, optimized spacing
- **Mobile** - Compact design, touch-friendly interface

## 🛠 Customization

### Custom App Name
```go
app := goryu.New(goryu.Config{
    AppName: "My Custom Application", // Shows in dashboard title
})
```

### Custom Monitoring Path
```go
app.EnableMonitoring("/admin/monitoring") // Custom base path
// UI available at: /admin/monitoring/ui
```

### Custom Health Checks
```go
app.AddHealthCheck("custom_service", &monitoring.HealthCheck{
    Check: func() (monitoring.HealthStatus, error) {
        // Your custom logic
        if serviceIsDown() {
            return monitoring.StatusUnhealthy, fmt.Errorf("service unavailable")
        }
        return monitoring.StatusHealthy, nil
    },
    Critical: false, // Won't fail overall health if this fails
})
```

## 📊 Event Types in UI

The dashboard displays different event types with distinct visual indicators:

- 🔵 **Request Events** - HTTP requests (blue dot)
- 🔴 **Error Events** - HTTP errors (red dot) 
- 🟣 **Custom Events** - Your application events (purple dot)
- 🟢 **Health Events** - Health check passes (green dot)
- 🟠 **Unhealthy Events** - Health check failures (orange dot)

## 🔄 Real-time Updates

The UI automatically refreshes data every 5 seconds. The system is designed to be:

- **Lightweight** - Minimal server impact
- **Efficient** - Only refreshes when tab is visible
- **Responsive** - No blocking or delays
- **Reliable** - Handles network issues gracefully

## 🎪 Example Dashboard Views

### Healthy Application
```
Status Overview: All Green
┌─────────────┬─────────────┬─────────────┬─────────────┐
│   Healthy   │  2h 30m 15s │    1,523    │      12     │
│ App Status  │   Uptime    │  Requests   │   Errors    │
└─────────────┴─────────────┴─────────────┴─────────────┘

Health Checks: All passing ✅
Recent Events: Mostly requests with few errors
Metrics: Normal memory usage, good response times
```

### Application with Issues
```
Status Overview: Degraded (Yellow)
┌─────────────┬─────────────┬─────────────┬─────────────┐
│  Degraded   │  45m 22s    │     892     │     127     │
│ App Status  │   Uptime    │  Requests   │   Errors    │
└─────────────┴─────────────┴─────────────┴─────────────┘

Health Checks: Database failing ❌, API degraded ⚠️
Recent Events: Multiple error events visible
Metrics: High error rate, elevated response times
```

## 🚀 Performance

The monitoring UI is designed to be:

- **Fast Loading** - Single HTML template, embedded CSS/JS
- **Lightweight** - No external dependencies
- **Efficient** - Minimal data transfer
- **Scalable** - Works well with high-traffic applications

## 🔧 Integration Examples

### With Docker
```dockerfile
EXPOSE 8080
CMD ["./app"]
# Dashboard available at http://container:8080/_monitor/ui
```

### With Load Balancer
```nginx
location /_monitor/ {
    proxy_pass http://app:8080/_monitor/;
    proxy_set_header Host $host;
}
```

### With Authentication
```go
// Protect monitoring endpoints
adminGroup := app.Group("/_monitor", authMiddleware())
// Then enable monitoring will respect the group middleware
```

## 💡 Best Practices

1. **Secure Access** - Protect monitoring endpoints in production
2. **Health Checks** - Add health checks for all critical dependencies
3. **Custom Events** - Emit events for important business operations
4. **Resource Monitoring** - Monitor memory, connections, etc.
5. **Alert Integration** - Use event handlers for alerting

## 🎨 UI Customization

The dashboard uses a modern design system:

- **Colors** - Blue gradient header, green/yellow/red status indicators
- **Typography** - System fonts for best performance
- **Layout** - CSS Grid for responsive design
- **Animations** - Subtle hover effects and transitions
- **Icons** - Color-coded dots for event types

## 📈 Metrics Displayed

### Request Metrics
- Total requests processed
- Error count and percentage
- Average response time
- Requests per second (calculated)

### System Metrics
- Memory usage (with human-readable formatting)
- Number of goroutines
- Application uptime
- Start time

### Health Metrics
- Overall health status
- Individual check status
- Check response times
- Critical vs non-critical indicators

This UI provides everything you need to monitor your Goryu applications effectively! 🎉