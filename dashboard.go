package main

import (
	"encoding/json"
	"net/http"
)

type BackendStatus struct {
	Address           string `json:"address"`
	Healthy           bool   `json:"healthy"`
	ActiveConnections int    `json:"active_connections"`
	CircuitState      string `json:"circuit_state"`
	TotalTripped      int64  `json:"total_tripped"`
}

type StatusResponse struct {
	TotalRequests int64           `json:"total_requests"`
	TotalErrors   int64           `json:"total_errors"`
	CacheHits     int64           `json:"cache_hits"`
	CacheMisses   int64           `json:"cache_misses"`
	Backends      []BackendStatus `json:"backends"`
}

func StatusAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	globalMetrics.mu.Lock()
	totReqs := globalMetrics.totalRequests
	totErrs := globalMetrics.totalErrors
	globalMetrics.mu.Unlock()

	mu.Lock()
	backendStatuses := make([]BackendStatus, 0, len(backends))
	for _, b := range backends {
		stateStr := "UNKNOWN"
		var tripped int64
		if b.CB != nil {
			stateStr = string(b.CB.State())
			tripped = b.CB.TotalTripped()
		}
		backendStatuses = append(backendStatuses, BackendStatus{
			Address:           b.Address,
			Healthy:           b.Healthy,
			ActiveConnections: b.ActiveConnections,
			CircuitState:      stateStr,
			TotalTripped:      tripped,
		})
	}
	mu.Unlock()

	resp := StatusResponse{
		TotalRequests: totReqs,
		TotalErrors:   totErrs,
		CacheHits:     globalCache.Hits(),
		CacheMisses:   globalCache.Misses(),
		Backends:      backendStatuses,
	}

	json.NewEncoder(w).Encode(resp)
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go Reverse Proxy Dashboard</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: 'Inter', sans-serif;
            background: #0f172a;
            color: #f8fafc;
            min-height: 100vh;
            padding: 2rem;
        }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2rem;
            padding-bottom: 1rem;
            border-bottom: 1px solid #1e293b;
        }
        .title-group { display: flex; align-items: center; gap: 0.75rem; }
        .logo { width: 32px; height: 32px; fill: #38bdf8; }
        h1 { font-size: 1.5rem; font-weight: 700; background: linear-gradient(135deg, #38bdf8, #818cf8); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
        .status-badge {
            background: rgba(34, 197, 94, 0.15);
            color: #4ade80;
            padding: 0.25rem 0.75rem;
            border-radius: 9999px;
            font-size: 0.875rem;
            font-weight: 500;
            border: 1px solid rgba(34, 197, 94, 0.3);
        }
        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
            gap: 1.25rem;
            margin-bottom: 2rem;
        }
        .card {
            background: rgba(30, 41, 59, 0.7);
            backdrop-filter: blur(12px);
            border: 1px solid #334155;
            border-radius: 12px;
            padding: 1.25rem;
            transition: transform 0.2s, border-color 0.2s;
        }
        .card:hover { transform: translateY(-2px); border-color: #475569; }
        .card-label { font-size: 0.875rem; color: #94a3b8; font-weight: 500; margin-bottom: 0.5rem; }
        .card-value { font-size: 1.875rem; font-weight: 700; color: #f8fafc; }
        .card-subtext { font-size: 0.75rem; color: #64748b; margin-top: 0.25rem; }
        
        .section-title { font-size: 1.25rem; font-weight: 600; margin-bottom: 1rem; color: #cbd5e1; }
        
        .backends-table {
            width: 100%;
            border-collapse: collapse;
            background: rgba(30, 41, 59, 0.7);
            backdrop-filter: blur(12px);
            border: 1px solid #334155;
            border-radius: 12px;
            overflow: hidden;
        }
        .backends-table th, .backends-table td {
            padding: 1rem 1.25rem;
            text-align: left;
            border-bottom: 1px solid #334155;
        }
        .backends-table th { background: #1e293b; color: #94a3b8; font-size: 0.875rem; font-weight: 600; }
        .backends-table tr:last-child td { border-bottom: none; }
        
        .badge {
            display: inline-block;
            padding: 0.2rem 0.6rem;
            border-radius: 6px;
            font-size: 0.75rem;
            font-weight: 600;
        }
        .badge-healthy { background: rgba(34, 197, 94, 0.2); color: #4ade80; border: 1px solid rgba(34, 197, 94, 0.3); }
        .badge-unhealthy { background: rgba(239, 68, 68, 0.2); color: #f87171; border: 1px solid rgba(239, 68, 68, 0.3); }
        
        .state-closed { color: #4ade80; }
        .state-open { color: #f87171; }
        .state-half-open { color: #fbbf24; }
    </style>
</head>
<body>
    <div class="header">
        <div class="title-group">
            <svg class="logo" viewBox="0 0 24 24"><path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/></svg>
            <h1>Go Reverse Proxy Observability</h1>
        </div>
        <div class="status-badge">Live Updates (2s)</div>
    </div>

    <div class="grid">
        <div class="card">
            <div class="card-label">Total Requests</div>
            <div class="card-value" id="tot-reqs">0</div>
            <div class="card-subtext">Processed HTTP connections</div>
        </div>
        <div class="card">
            <div class="card-label">Total Errors</div>
            <div class="card-value" id="tot-errs" style="color: #f87171;">0</div>
            <div class="card-subtext">5xx / 4xx error count</div>
        </div>
        <div class="card">
            <div class="card-label">Cache Hit Ratio</div>
            <div class="card-value" id="cache-ratio">0.0%</div>
            <div class="card-subtext" id="cache-sub">Hits: 0 | Misses: 0</div>
        </div>
        <div class="card">
            <div class="card-label">Active Backends</div>
            <div class="card-value" id="active-backends">0/0</div>
            <div class="card-subtext">Healthy pool nodes</div>
        </div>
    </div>

    <div class="section-title">Backend Pool Cluster</div>
    <table class="backends-table">
        <thead>
            <tr>
                <th>Backend Address</th>
                <th>Health Status</th>
                <th>Active Connections</th>
                <th>Circuit Breaker</th>
                <th>Total Trips</th>
            </tr>
        </thead>
        <tbody id="backends-body">
            <tr><td colspan="5" style="text-align: center; color: #64748b;">Loading cluster state...</td></tr>
        </tbody>
    </table>

    <script>
        async function fetchStatus() {
            try {
                const res = await fetch('/api/status');
                const data = await res.json();
                
                document.getElementById('tot-reqs').textContent = data.total_requests.toLocaleString();
                document.getElementById('tot-errs').textContent = data.total_errors.toLocaleString();
                
                const totalCache = data.cache_hits + data.cache_misses;
                const ratio = totalCache > 0 ? ((data.cache_hits / totalCache) * 100).toFixed(1) : '0.0';
                document.getElementById('cache-ratio').textContent = ratio + '%';
                document.getElementById('cache-sub').textContent = 'Hits: ' + data.cache_hits + ' | Misses: ' + data.cache_misses;
                
                const healthyCount = data.backends.filter(function(b) { return b.healthy; }).length;
                document.getElementById('active-backends').textContent = healthyCount + '/' + data.backends.length;
                
                const tbody = document.getElementById('backends-body');
                tbody.innerHTML = data.backends.map(function(b) {
                    const healthBadge = b.healthy ? 
                        '<span class="badge badge-healthy">HEALTHY</span>' : 
                        '<span class="badge badge-unhealthy">UNHEALTHY</span>';
                    
                    let stateClass = 'state-closed';
                    if (b.circuit_state === 'OPEN') stateClass = 'state-open';
                    if (b.circuit_state === 'HALF-OPEN') stateClass = 'state-half-open';
                    
                    return '<tr>' +
                        '<td style="font-weight: 600;">' + b.address + '</td>' +
                        '<td>' + healthBadge + '</td>' +
                        '<td>' + b.active_connections + '</td>' +
                        '<td class="' + stateClass + '" style="font-weight: 600;">' + b.circuit_state + '</td>' +
                        '<td>' + b.total_tripped + '</td>' +
                    '</tr>';
                }).join('');
            } catch (err) {
                console.error('Failed to fetch status:', err);
            }
        }
        fetchStatus();
        setInterval(fetchStatus, 2000);
    </script>
</body>
</html>`

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(dashboardHTML))
}
