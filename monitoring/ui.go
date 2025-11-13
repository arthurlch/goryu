package monitoring

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/arthurlch/goryu/context"
)

const dashboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.AppName}} - Monitoring Dashboard</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: #f5f7fa;
            color: #333;
            line-height: 1.6;
        }

        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 1.5rem 0;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 0 20px;
        }

        .header h1 {
            font-size: 2rem;
            font-weight: 600;
            margin-bottom: 0.5rem;
        }

        .header .subtitle {
            font-size: 1rem;
            opacity: 0.9;
        }

        .main-content {
            padding: 2rem 0;
        }

        .status-overview {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2rem;
        }

        .status-card {
            background: white;
            padding: 1.5rem;
            border-radius: 12px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.08);
            border-left: 4px solid #667eea;
            transition: transform 0.2s;
        }

        .status-card:hover {
            transform: translateY(-2px);
        }

        .status-card.healthy {
            border-left-color: #10b981;
        }

        .status-card.degraded {
            border-left-color: #f59e0b;
        }

        .status-card.unhealthy {
            border-left-color: #ef4444;
        }

        .status-card h3 {
            font-size: 0.875rem;
            color: #64748b;
            text-transform: uppercase;
            font-weight: 600;
            letter-spacing: 0.05em;
            margin-bottom: 0.5rem;
        }

        .status-card .value {
            font-size: 2rem;
            font-weight: 700;
            color: #1e293b;
            margin-bottom: 0.25rem;
        }

        .status-card .description {
            font-size: 0.875rem;
            color: #64748b;
        }

        .dashboard-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 2rem;
            margin-bottom: 2rem;
        }

        .panel {
            background: white;
            border-radius: 12px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.08);
            overflow: hidden;
        }

        .panel-header {
            background: #f8fafc;
            padding: 1rem 1.5rem;
            border-bottom: 1px solid #e2e8f0;
        }

        .panel-header h2 {
            font-size: 1.125rem;
            font-weight: 600;
            color: #1e293b;
        }

        .panel-content {
            padding: 1.5rem;
        }

        .health-check {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0.75rem 0;
            border-bottom: 1px solid #f1f5f9;
        }

        .health-check:last-child {
            border-bottom: none;
        }

        .health-check-name {
            font-weight: 500;
            color: #1e293b;
        }

        .health-check-details {
            font-size: 0.875rem;
            color: #64748b;
        }

        .status-badge {
            padding: 0.25rem 0.75rem;
            border-radius: 9999px;
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.025em;
        }

        .status-badge.healthy {
            background: #dcfce7;
            color: #166534;
        }

        .status-badge.degraded {
            background: #fef3c7;
            color: #92400e;
        }

        .status-badge.unhealthy {
            background: #fecaca;
            color: #991b1b;
        }

        .event-item {
            display: flex;
            align-items: flex-start;
            gap: 1rem;
            padding: 1rem 0;
            border-bottom: 1px solid #f1f5f9;
        }

        .event-item:last-child {
            border-bottom: none;
        }

        .event-icon {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            margin-top: 0.5rem;
            flex-shrink: 0;
        }

        .event-icon.request {
            background: #3b82f6;
        }

        .event-icon.error {
            background: #ef4444;
        }

        .event-icon.custom {
            background: #8b5cf6;
        }

        .event-icon.healthy {
            background: #10b981;
        }

        .event-icon.unhealthy {
            background: #ef4444;
        }

        .event-content {
            flex: 1;
        }

        .event-message {
            font-weight: 500;
            color: #1e293b;
            margin-bottom: 0.25rem;
        }

        .event-details {
            font-size: 0.875rem;
            color: #64748b;
        }

        .event-time {
            font-size: 0.75rem;
            color: #94a3b8;
            margin-top: 0.25rem;
        }

        .refresh-info {
            background: #f1f5f9;
            padding: 1rem;
            border-radius: 8px;
            text-align: center;
            margin-top: 2rem;
            font-size: 0.875rem;
            color: #64748b;
        }

        .loading {
            display: inline-block;
            width: 16px;
            height: 16px;
            border: 2px solid #f3f4f6;
            border-top: 2px solid #667eea;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }

        @media (max-width: 768px) {
            .dashboard-grid {
                grid-template-columns: 1fr;
            }
            
            .status-overview {
                grid-template-columns: repeat(2, 1fr);
            }
            
            .container {
                padding: 0 15px;
            }
        }

        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 1rem;
        }

        .metric-item {
            text-align: center;
            padding: 1rem;
            background: #f8fafc;
            border-radius: 8px;
        }

        .metric-value {
            font-size: 1.5rem;
            font-weight: 700;
            color: #1e293b;
            margin-bottom: 0.25rem;
        }

        .metric-label {
            font-size: 0.75rem;
            color: #64748b;
            text-transform: uppercase;
            font-weight: 600;
            letter-spacing: 0.05em;
        }

        .full-width {
            grid-column: 1 / -1;
            margin-top: 2rem;
        }
    </style>
</head>
<body>
    <header class="header">
        <div class="container">
            <h1>{{.AppName}} Monitoring</h1>
            <p class="subtitle">Real-time application health and performance dashboard</p>
        </div>
    </header>

    <main class="main-content">
        <div class="container">
            <!-- Status Overview -->
            <div class="status-overview">
                <div class="status-card {{.Status}}">
                    <h3>Overall Status</h3>
                    <div class="value">{{.Status | title}}</div>
                    <div class="description">Application health</div>
                </div>
                <div class="status-card">
                    <h3>Uptime</h3>
                    <div class="value">{{.Metrics.Uptime | formatDuration}}</div>
                    <div class="description">Since {{.Metrics.StartTime | formatTime}}</div>
                </div>
                <div class="status-card">
                    <h3>Requests</h3>
                    <div class="value">{{.Metrics.RequestCount}}</div>
                    <div class="description">Total processed</div>
                </div>
                <div class="status-card">
                    <h3>Errors</h3>
                    <div class="value">{{.Metrics.ErrorCount}}</div>
                    <div class="description">Error responses</div>
                </div>
            </div>

            <!-- Main Dashboard Grid -->
            <div class="dashboard-grid">
                <!-- Health Checks Panel -->
                <div class="panel">
                    <div class="panel-header">
                        <h2>Health Checks</h2>
                    </div>
                    <div class="panel-content">
                        {{if .HealthChecks}}
                            {{range $name, $check := .HealthChecks}}
                            <div class="health-check">
                                <div>
                                    <div class="health-check-name">{{$name}}{{if $check.Critical}} (Critical){{end}}</div>
                                    <div class="health-check-details">
                                        Duration: {{$check.Duration | formatDuration}}
                                        {{if $check.Message}}<br>{{$check.Message}}{{end}}
                                    </div>
                                </div>
                                <span class="status-badge {{$check.Status}}">{{$check.Status}}</span>
                            </div>
                            {{end}}
                        {{else}}
                            <p style="color: #64748b; text-align: center;">No health checks configured</p>
                        {{end}}
                    </div>
                </div>

                <!-- Recent Events Panel -->
                <div class="panel">
                    <div class="panel-header">
                        <h2>Recent Events</h2>
                    </div>
                    <div class="panel-content">
                        {{if .Events}}
                            {{range .Events}}
                            <div class="event-item">
                                <div class="event-icon {{.Type}}"></div>
                                <div class="event-content">
                                    <div class="event-message">{{.Message}}</div>
                                    {{if .Data}}
                                    <div class="event-details">
                                        {{if .Data.status_code}}Status: {{.Data.status_code}} | {{end}}
                                        {{if .Data.duration_ms}}Duration: {{.Data.duration_ms}}ms{{end}}
                                    </div>
                                    {{end}}
                                    <div class="event-time">{{.Timestamp | formatTime}}</div>
                                </div>
                            </div>
                            {{end}}
                        {{else}}
                            <p style="color: #64748b; text-align: center;">No recent events</p>
                        {{end}}
                    </div>
                </div>
            </div>

            <!-- Metrics Panel (Full Width) -->
            <div class="panel full-width">
                <div class="panel-header">
                    <h2>System Metrics</h2>
                </div>
                <div class="panel-content">
                    <div class="metrics-grid">
                        <div class="metric-item">
                            <div class="metric-value">{{.Metrics.MemoryUsage | formatBytes}}</div>
                            <div class="metric-label">Memory Usage</div>
                        </div>
                        <div class="metric-item">
                            <div class="metric-value">{{.Metrics.GoRoutines}}</div>
                            <div class="metric-label">Goroutines</div>
                        </div>
                        <div class="metric-item">
                            <div class="metric-value">{{.Metrics.AvgResponseTime | formatDuration}}</div>
                            <div class="metric-label">Avg Response</div>
                        </div>
                        <div class="metric-item">
                            <div class="metric-value">{{if .Metrics.RequestCount}}{{printf "%.2f" (div .Metrics.ErrorCount .Metrics.RequestCount | mul 100)}}%{{else}}0%{{end}}</div>
                            <div class="metric-label">Error Rate</div>
                        </div>
                    </div>
                </div>
            </div>

            <div class="refresh-info">
                <span class="loading"></span>
                <span style="margin-left: 0.5rem;">Auto-refreshing every 5 seconds</span>
            </div>
        </div>
    </main>

    <script>
        // Auto-refresh functionality
        let refreshInterval;
        
        function startAutoRefresh() {
            refreshInterval = setInterval(() => {
                window.location.reload();
            }, 5000);
        }

        function stopAutoRefresh() {
            if (refreshInterval) {
                clearInterval(refreshInterval);
            }
        }

        // Handle visibility change to pause/resume refresh when tab is hidden
        document.addEventListener('visibilitychange', function() {
            if (document.hidden) {
                stopAutoRefresh();
            } else {
                startAutoRefresh();
            }
        });

        // Start auto-refresh when page loads
        startAutoRefresh();

        // Optional: Add click handlers for manual refresh
        document.addEventListener('keydown', function(e) {
            if (e.key === 'r' || e.key === 'R') {
                window.location.reload();
            }
        });

        // Add some interactivity to status cards
        document.querySelectorAll('.status-card, .panel').forEach(card => {
            card.addEventListener('mouseenter', function() {
                this.style.boxShadow = '0 4px 20px rgba(0,0,0,0.15)';
            });
            
            card.addEventListener('mouseleave', function() {
                this.style.boxShadow = '0 2px 10px rgba(0,0,0,0.08)';
            });
        });

        // Show loading state during refresh
        let isRefreshing = false;
        
        window.addEventListener('beforeunload', function() {
            isRefreshing = true;
            document.querySelector('.loading').style.opacity = '1';
        });
    </script>
</body>
</html>
`

func (m *Monitor) UIHandler(appName string) context.HandlerFunc {
	tmpl := template.New("dashboard").Funcs(template.FuncMap{
		"title": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return string(s[0]-32) + s[1:]
		},
		"formatDuration": func(d time.Duration) string {
			if d < time.Minute {
				return d.Truncate(time.Second).String()
			}
			if d < time.Hour {
				return d.Truncate(time.Minute).String()
			}
			if d < 24*time.Hour {
				return d.Truncate(time.Hour).String()
			}
			days := int(d.Hours() / 24)
			hours := int(d.Hours()) % 24
			if days > 0 {
				return fmt.Sprintf("%dd %dh", days, hours)
			}
			return d.Truncate(time.Hour).String()
		},
		"formatTime": func(t time.Time) string {
			return t.Format("15:04:05")
		},
		"formatBytes": func(bytes uint64) string {
			const unit = 1024
			if bytes < unit {
				return fmt.Sprintf("%d B", bytes)
			}
			div, exp := uint64(unit), 0
			for n := bytes / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
		},
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"div": func(a, b int64) float64 {
			if b == 0 {
				return 0
			}
			return float64(a) / float64(b)
		},
	})

	template.Must(tmpl.Parse(dashboardHTML))

	return func(c *context.Context) {
		if appName == "" {
			appName = "Application"
		}

		status := m.GetHealthStatus()
		metrics := m.GetMetrics()
		healthChecks := m.GetHealthResults()
		events := m.GetEvents(10) // Get last 10 events

		data := struct {
			AppName      string
			Status       string
			Metrics      *Metrics
			HealthChecks map[string]*HealthResult
			Events       []Event
		}{
			AppName:      appName,
			Status:       string(status),
			Metrics:      metrics,
			HealthChecks: healthChecks,
			Events:       events,
		}

		c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(c.Writer, data); err != nil {
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
