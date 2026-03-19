package dashboard

import (
	"net/http"

	authdomain "github.com/trungtran/coder/internal/domain/auth"
)

type overviewData struct {
	Version string
	Page    string
	Stats   authdomain.ActivityStats
}

// handleOverview serves GET /dashboard/overview.
func (d *DashboardServer) handleOverview(w http.ResponseWriter, r *http.Request) {
	stats, err := d.mgr.GetActivityStats(r.Context(), 30)
	if err != nil {
		http.Error(w, "failed to load stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := overviewData{
		Version: d.version,
		Page:    "overview",
		Stats:   stats,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := d.tmpl.ExecuteTemplate(w, "overview.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleStatsChart serves GET /dashboard/stats/chart — JSON for line chart.
func (d *DashboardServer) handleStatsChart(w http.ResponseWriter, r *http.Request) {
	stats, err := d.mgr.GetActivityStats(r.Context(), 30)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Build a simple JSON object with labels and values arrays.
	w.Write([]byte(`{"labels":[`))
	for i, dc := range stats.CommandsPerDay {
		if i > 0 {
			w.Write([]byte(","))
		}
		w.Write([]byte(`"` + dc.Date + `"`))
	}
	w.Write([]byte(`],"values":[`))
	for i, dc := range stats.CommandsPerDay {
		if i > 0 {
			w.Write([]byte(","))
		}
		w.Write([]byte(itoa(dc.Count)))
	}
	w.Write([]byte(`]}`))
}

// handleStatsCommands serves GET /dashboard/stats/commands — JSON for donut chart.
func (d *DashboardServer) handleStatsCommands(w http.ResponseWriter, r *http.Request) {
	stats, err := d.mgr.GetActivityStats(r.Context(), 30)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"commands":[`))
	for i, cc := range stats.TopCommands {
		if i > 0 {
			w.Write([]byte(","))
		}
		w.Write([]byte(`{"command":"` + jsonEscape(cc.Command) + `","count":` + itoa(cc.Count) + `}`))
	}
	w.Write([]byte(`]}`))
}

// itoa converts an int to its decimal string representation without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

// jsonEscape escapes a string for safe inclusion in a JSON string literal.
func jsonEscape(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			out = append(out, '\\', '"')
		case '\\':
			out = append(out, '\\', '\\')
		case '\n':
			out = append(out, '\\', 'n')
		case '\r':
			out = append(out, '\\', 'r')
		case '\t':
			out = append(out, '\\', 't')
		default:
			out = append(out, c)
		}
	}
	return string(out)
}
