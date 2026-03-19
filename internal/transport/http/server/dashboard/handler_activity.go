package dashboard

import (
	"net/http"
	"strconv"

	authdomain "github.com/trungtran/coder/internal/domain/auth"
)

type activityData struct {
	Version    string
	Page       string
	Activities []authdomain.Activity
	Clients    []authdomain.Client
	HasMore    bool
	NextOffset int
	Filter     authdomain.ActivityFilter
	Charts     authdomain.ActivityChartStats
}

const defaultActivityLimit = 50
const defaultActivityChartDays = 30

// handleActivity serves GET /dashboard/activity with pagination.
// Full page on first load; only tbody rows partial when HX-Request header is set.
func (d *DashboardServer) handleActivity(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	filter := authdomain.ActivityFilter{
		ClientID: q.Get("client_id"),
		Command:  q.Get("command"),
		Limit:    defaultActivityLimit,
		Offset:   offset,
	}

	activities, total, err := d.mgr.GetAllActivities(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to load activities: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if activities == nil {
		activities = []authdomain.Activity{}
	}

	nextOffset := offset + defaultActivityLimit
	hasMore := nextOffset < total

	clients, _ := d.mgr.ListClients(r.Context())
	if clients == nil {
		clients = []authdomain.Client{}
	}

	isHTMX := r.Header.Get("HX-Request") == "true"

	// For HTMX partial requests (row pagination / filter change), skip chart data.
	if isHTMX {
		data := activityData{
			Version:    d.version,
			Page:       "activity",
			Activities: activities,
			Clients:    clients,
			HasMore:    hasMore,
			NextOffset: nextOffset,
			Filter:     filter,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := d.tmpl.ExecuteTemplate(w, "activity_rows", data); err != nil {
			http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Full page load — also compute chart data for the Charts tab.
	charts, _ := d.mgr.GetActivityChartStats(r.Context(), filter, defaultActivityChartDays)

	data := activityData{
		Version:    d.version,
		Page:       "activity",
		Activities: activities,
		Clients:    clients,
		HasMore:    hasMore,
		NextOffset: nextOffset,
		Filter:     filter,
		Charts:     charts,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := d.tmpl.ExecuteTemplate(w, "activity.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}
