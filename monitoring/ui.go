package monitoring

import (
	"html/template"
	"net/http"

	context "github.com/arthurlch/goryu/goryuctx"
)

const dashboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Goryu Monitoring</title>
    <style>
        :root {
            --primary: #6366f1;
            --primary-dark: #4f46e5;
            --success: #10b981;
            --warning: #f59e0b;
            --danger: #ef4444;
            --bg: #0f172a;
            --surface: #1e293b;
            --surface-hover: #334155;
            --text: #f8fafc;
            --text-muted: #94a3b8;
            --border: #334155;
        }

        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background-color: var(--bg);
            color: var(--text);
            line-height: 1.5;
            height: 100vh;
            display: flex;
            flex-direction: column;
        }

        /* Utils */
        .container { max-width: 1400px; margin: 0 auto; padding: 0 1.5rem; width: 100%; }
        .flex { display: flex; }
        .items-center { align-items: center; }
        .justify-between { justify-content: space-between; }
        .gap-2 { gap: 0.5rem; }
        .gap-4 { gap: 1rem; }
        .grid { display: grid; gap: 1.5rem; }
        .grid-cols-2 { grid-template-columns: repeat(2, 1fr); }
        .grid-cols-4 { grid-template-columns: repeat(4, 1fr); }
        
        /* Typography */
        h1 { font-size: 1.5rem; font-weight: 700; letter-spacing: -0.025em; }
        h2 { font-size: 1.1rem; font-weight: 600; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 1rem; }
        .text-sm { font-size: 0.875rem; }
        .text-muted { color: var(--text-muted); }
        .font-mono { font-family: 'JetBrains Mono', monospace; }

        /* Helpers */
        .status-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; margin-right: 6px; }
        .status-dot.healthy { background-color: var(--success); box-shadow: 0 0 8px rgba(16, 185, 129, 0.4); }
        .status-dot.unhealthy { background-color: var(--danger); box-shadow: 0 0 8px rgba(239, 68, 68, 0.4); }
        .status-dot.degraded { background-color: var(--warning); }

        .badge { padding: 2px 8px; border-radius: 4px; font-size: 0.7rem; font-weight: 600; text-transform: uppercase; }
        .badge.GET { background: rgba(99, 102, 241, 0.2); color: #818cf8; }
        .badge.POST { background: rgba(16, 185, 129, 0.2); color: #34d399; }
        .badge.PUT { background: rgba(245, 158, 11, 0.2); color: #fbbf24; }
        .badge.DELETE { background: rgba(239, 68, 68, 0.2); color: #f87171; }

        /* Layout */
        header {
            background-color: rgba(30, 41, 59, 0.8);
            backdrop-filter: blur(8px);
            border-bottom: 1px solid var(--border);
            padding: 1rem 0;
            position: sticky;
            top: 0;
            z-index: 100;
        }

        main { padding: 2rem 0; flex: 1; overflow-y: auto; }

        /* Cards */
        .card {
            background-color: var(--surface);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 1.5rem;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .card:hover { transform: translateY(-2px); box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.3); border-color: var(--primary); }

        .stat-value { font-size: 2.25rem; font-weight: 800; color: var(--text); line-height: 1; margin: 0.5rem 0; }
        .stat-label { font-size: 0.875rem; color: var(--text-muted); font-weight: 500; }

        /* Events List */
        .events-panel { grid-column: span 2; display: flex; flex-direction: column; height: 600px; }
        .events-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
        
        .search-box {
            background: var(--bg);
            border: 1px solid var(--border);
            color: var(--text);
            padding: 0.5rem 1rem;
            border-radius: 6px;
            width: 300px;
            font-size: 0.875rem;
            outline: none;
            transition: all 0.2s;
        }
        .search-box:focus { border-color: var(--primary); box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2); }

        .events-list {
            flex: 1;
            overflow-y: auto;
            border: 1px solid var(--border);
            border-radius: 8px;
            background: var(--bg);
        }

        .event-row {
            padding: 0.75rem 1rem;
            border-bottom: 1px solid var(--border);
            display: grid;
            grid-template-columns: 100px 80px 1fr auto;
            gap: 1rem;
            align-items: center;
            font-size: 0.875rem;
            transition: background 0.1s;
        }
        .event-row:hover { background: var(--surface-hover); }
        .event-row:last-child { border-bottom: none; }
        
        .event-time { color: var(--text-muted); font-family: monospace; font-size: 0.8rem; }
        .event-type { font-weight: 600; font-size: 0.75rem; text-transform: uppercase; }
        .event-type.error { color: var(--danger); }
        .event-type.request { color: var(--primary); }
        .event-type.custom { color: var(--success); }

        /* Animations */
        @keyframes pulse {
            0% { opacity: 1; }
            50% { opacity: 0.5; }
            100% { opacity: 1; }
        }
        .live-indicator {
            display: flex;
            align-items: center;
            font-size: 0.75rem;
            color: var(--success);
            font-weight: 600;
        }
        .live-dot {
            width: 6px; height: 6px; background: var(--success); border-radius: 50%;
            margin-right: 6px; animation: pulse 2s infinite;
        }

        /* Responsive */
        @media (max-width: 1024px) {
            .grid-cols-4 { grid-template-columns: repeat(2, 1fr); }
        }
        @media (max-width: 768px) {
            .grid-cols-2, .grid-cols-4 { grid-template-columns: 1fr; }
            .events-panel { grid-column: span 1; }
            .event-row { grid-template-columns: 1fr; gap: 0.25rem; }
        }
        /* Toggle Chips */
        .filter-chips { display: flex; gap: 0.5rem; margin-right: 1rem; }
        .chip { 
            padding: 0.35rem 0.75rem; 
            border-radius: 999px; 
            font-size: 0.75rem; 
            font-weight: 600; 
            cursor: pointer; 
            border: 1px solid var(--border);
            color: var(--text-muted);
            background: var(--surface);
            transition: all 0.2s;
            user-select: none;
        }
        .chip:hover { border-color: var(--primary); color: var(--text); }
        .chip.active { background: var(--primary); color: white; border-color: var(--primary); }
        .chip.active.error { background: var(--danger); border-color: var(--danger); }
        .chip.active.request { background: var(--primary); border-color: var(--primary); }
        .chip.active.custom { background: var(--success); border-color: var(--success); }
    </style>
</head>
<body>
    <header>
        <div class="container flex justify-between items-center">
            <div class="flex items-center gap-4">
                <div style="background: var(--primary); width: 32px; height: 32px; border-radius: 8px; display: grid; place-items: center; font-weight: bold;">G</div>
                <h1>{{.AppName}}</h1>
            </div>
            <div class="live-indicator">
                <div class="live-dot"></div>
                LIVE
            </div>
        </div>
    </header>

    <main>
        <div class="container grid">
            <!-- Top Stats -->
            <div class="grid grid-cols-4">
                <div class="card">
                    <div class="stat-label">Health Status</div>
                    <div class="stat-value flex items-center" id="health-status">
                        <span class="status-dot"></span>
                        <span id="health-text">--</span>
                    </div>
                </div>
                <div class="card">
                    <div class="stat-label">Total Requests</div>
                    <div class="stat-value" id="req-count">0</div>
                </div>
                <div class="card">
                    <div class="stat-label">Total Errors</div>
                    <div class="stat-value" style="color: var(--danger);" id="err-count">0</div>
                </div>
                <div class="card">
                    <div class="stat-label">Avg Latency</div>
                    <div class="stat-value" id="avg-latency">0ms</div>
                </div>
            </div>

            <div class="grid grid-cols-2">
                <!-- Health Checks -->
                <div class="card">
                    <h2>System Health</h2>
                    <div id="health-checks-list" class="grid" style="gap: 1rem;">
                        <!-- Injected via JS -->
                    </div>
                </div>

                <!-- Metrics Detail -->
                <div class="card">
                    <h2>Runtime Metrics</h2>
                    <div class="grid grid-cols-2" style="gap: 2rem;">
                        <div>
                            <div class="stat-label">Goroutines</div>
                            <div class="stat-value" style="font-size: 1.5rem;" id="goroutines">0</div>
                        </div>
                        <div>
                            <div class="stat-label">Memory Usage</div>
                            <div class="stat-value" style="font-size: 1.5rem;" id="memory">0 MB</div>
                        </div>
                        <div>
                            <div class="stat-label">Uptime</div>
                            <div class="stat-value" style="font-size: 1.5rem;" id="uptime">0s</div>
                        </div>
                    </div>
                </div>
             
                <!-- Events Log -->
                <div class="card events-panel">
                    <div class="events-header">
                        <h2>Event Log</h2>
                        <div class="flex items-center">
                            <div class="filter-chips" id="filter-chips">
                                <span class="chip active request" data-type="request">Requests</span>
                                <span class="chip active error" data-type="error">Errors</span>
                                <span class="chip active custom" data-type="custom">Custom</span>
                            </div>
                            <input type="text" id="event-filter" class="search-box" placeholder="Search events...">
                        </div>
                    </div>
                    <div class="events-list" id="events-list">
                        <!-- Injected via JS -->
                    </div>
                </div>
            </div>
        </div>
    </main>

    <script>
        const state = {
            events: [],
            filter: '',
            activeTypes: new Set(['request', 'error', 'custom'])
        };

        const els = {
            healthStatus: document.getElementById('health-status'),
            healthText: document.getElementById('health-text'),
            reqCount: document.getElementById('req-count'),
            errCount: document.getElementById('err-count'),
            avgLatency: document.getElementById('avg-latency'),
            healthChecksList: document.getElementById('health-checks-list'),
            goroutines: document.getElementById('goroutines'),
            memory: document.getElementById('memory'),
            uptime: document.getElementById('uptime'),
            eventsList: document.getElementById('events-list'),
            eventFilter: document.getElementById('event-filter')
        };

        // Formatters
        const formatBytes = (bytes) => {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        };

        const formatTime = (ts) => {
            return new Date(ts).toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
        };

        async function fetchData() {
            try {
                const [healthRes, metricsRes, eventsRes] = await Promise.all([
                    fetch('/_health').then(r => r.json()),
                    fetch('/_metrics').then(r => r.json()),
                    fetch('/_events?limit=100').then(r => r.json())
                ]);

                updateHealth(healthRes);
                updateMetrics(metricsRes);
                updateEvents(eventsRes.events);
            } catch (e) {
                console.error("Monitoring fetch failed", e);
            }
        }

        function updateHealth(data) {
            const statusColor = data.status === 'healthy' ? 'healthy' : (data.status === 'degraded' ? 'degraded' : 'unhealthy');
            els.healthText.textContent = data.status.toUpperCase();
            els.healthText.className = "text-" + statusColor;
            els.healthStatus.querySelector('.status-dot').className = "status-dot " + statusColor;

            // Health Checks List
            els.healthChecksList.innerHTML = Object.entries(data.checks || {}).map(([name, check]) => {
                const msgHtml = check.message ? '<div class="text-sm text-danger">' + check.message + '</div>' : '';
                return '<div class="flex justify-between items-center p-2 rounded" style="background: var(--bg);">' +
                    '<div>' +
                        '<div class="font-bold capitalize">' + name + '</div>' +
                        '<div class="text-sm text-muted">' + (check.duration / 1000000).toFixed(2) + 'ms</div>' +
                        msgHtml +
                    '</div>' +
                    '<span class="status-dot ' + (check.status === 'healthy' ? 'healthy' : 'unhealthy') + '"></span>' +
                '</div>';
            }).join('');
        }

        function updateMetrics(data) {
            els.reqCount.textContent = data.request_count.toLocaleString();
            els.errCount.textContent = data.error_count.toLocaleString();
            // Convert ns to ms
            els.avgLatency.textContent = (data.avg_response_time / 1000000).toFixed(1) + 'ms';
            els.goroutines.textContent = data.goroutines;
            els.memory.textContent = formatBytes(data.memory_usage_bytes);
            
            // Simple uptime formatting
            const uptimeSecs = data.uptime / 1000000000;
            const h = Math.floor(uptimeSecs / 3600);
            const m = Math.floor((uptimeSecs % 3600) / 60);
            els.uptime.textContent = h + 'h ' + m + 'm';
        }

        function updateEvents(newEvents) {
            if (!newEvents) return;
            // Reverse to show newest top, locally
            state.events = newEvents.reverse(); 
            renderEvents();
        }

        function renderEvents() {
            const filter = state.filter.toLowerCase();
            const filtered = state.events.filter(e => {
                // Type Filter
                if (!state.activeTypes.has(e.type)) return false;

                // Text Filter
                return e.message.toLowerCase().includes(filter) || 
                       e.type.toLowerCase().includes(filter) ||
                       (e.data && JSON.stringify(e.data).toLowerCase().includes(filter));
            });

            els.eventsList.innerHTML = filtered.map(e => {
                const dataHtml = e.data ? '<div class="text-sm text-muted font-mono" style="margin-top:2px;">' + JSON.stringify(e.data).replace(/["{}]/g, "") + '</div>' : '';
                const durationHtml = e.duration ? (e.duration/1000000).toFixed(1) + 'ms' : '';
                
                return '<div class="event-row">' +
                    '<div class="event-time">' + formatTime(e.timestamp) + '</div>' +
                    '<div class="event-type ' + e.type + '">' + e.type + '</div>' +
                    '<div class="event-msg">' +
                        e.message +
                        dataHtml +
                    '</div>' +
                    '<div class="text-muted text-sm">' + durationHtml + '</div>' +
                '</div>';
            }).join('');
        }

        // Filter Chips Logic
        document.querySelectorAll('.chip').forEach(chip => {
            chip.addEventListener('click', (e) => {
                const type = e.target.dataset.type;
                if (state.activeTypes.has(type)) {
                    state.activeTypes.delete(type);
                    e.target.classList.remove('active');
                } else {
                    state.activeTypes.add(type);
                    e.target.classList.add('active');
                }
                renderEvents();
            });
        });

        els.eventFilter.addEventListener('input', (e) => {
            state.filter = e.target.value;
            renderEvents();
        });

        // Loop
        fetchData();
        setInterval(fetchData, 2000); // 2s polling
    </script>
</body>
</html>
`

func (m *Monitor) UIHandler(appName string) context.HandlerFunc {
	if appName == "" {
		appName = "Goryu App"
	}
	
	// Create template once
	tmpl, err := template.New("dashboard").Parse(dashboardHTML)
	if err != nil {
		panic(err) // Should be safe as template is const
	}

	return func(c *context.Context) {
		c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		
		data := struct {
			AppName string
		}{
			AppName: appName,
		}

		if err := tmpl.Execute(c.Writer, data); err != nil {
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
