package dashboard

import (
	"net/http"
	"strings"

	authdomain "github.com/trungtran/coder/internal/domain/auth"
)

type clientsData struct {
	Version string
	Page    string
	Clients []authdomain.Client
}

type clientDrawerData struct {
	ClientEmail string
	Activities  []authdomain.Activity
}

// handleClients serves GET /dashboard/clients.
func (d *DashboardServer) handleClients(w http.ResponseWriter, r *http.Request) {
	clients, err := d.mgr.ListClients(r.Context())
	if err != nil {
		http.Error(w, "failed to list clients: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if clients == nil {
		clients = []authdomain.Client{}
	}

	data := clientsData{
		Version: d.version,
		Page:    "clients",
		Clients: clients,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := d.tmpl.ExecuteTemplate(w, "clients.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleClientDetail routes /dashboard/clients/{id} and /dashboard/clients/{id}/activity.
func (d *DashboardServer) handleClientDetail(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/dashboard/clients/")
	parts := strings.SplitN(trimmed, "/", 2)
	clientID := parts[0]
	if clientID == "" {
		http.NotFound(w, r)
		return
	}

	if len(parts) > 1 && parts[1] == "activity" {
		d.handleClientActivity(w, r, clientID)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		d.revokeClient(w, r, clientID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// revokeClient handles DELETE /dashboard/clients/{id}.
func (d *DashboardServer) revokeClient(w http.ResponseWriter, r *http.Request, clientID string) {
	if err := d.mgr.RevokeClient(r.Context(), clientID); err != nil {
		http.Error(w, "failed to revoke client: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Return empty response — HTMX will remove the target element.
	w.Header().Set("HX-Trigger", `{"showToast":"Client revoked."}`)
	w.WriteHeader(http.StatusOK)
}

// handleClientActivity serves GET /dashboard/clients/{id}/activity — drawer partial.
func (d *DashboardServer) handleClientActivity(w http.ResponseWriter, r *http.Request, clientID string) {
	filter := authdomain.ActivityFilter{
		ClientID: clientID,
		Limit:    20,
		Offset:   0,
	}
	activities, _, err := d.mgr.GetAllActivities(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to load activities: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Determine the client email from the first activity or fall back to clientID.
	email := clientID
	if len(activities) > 0 && activities[0].GitEmail != "" {
		email = activities[0].GitEmail
	}

	data := clientDrawerData{
		ClientEmail: email,
		Activities:  activities,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := d.tmpl.ExecuteTemplate(w, "activity_drawer", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}
