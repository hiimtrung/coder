package dashboard

import (
	"net/http"
)

type settingsData struct {
	Version             string
	Page                string
	BootstrapConfigured bool
	SecureMode          bool
}

type tokenResultData struct {
	Token string
}

// handleSettings serves GET /dashboard/settings.
func (d *DashboardServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	configured, _ := d.mgr.HasBootstrapToken(r.Context())

	data := settingsData{
		Version:             d.version,
		Page:                "settings",
		BootstrapConfigured: configured,
		SecureMode:          d.mgr.IsSecureMode(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := d.tmpl.ExecuteTemplate(w, "settings.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleSettingsRegenerate handles POST /dashboard/settings/regenerate.
// Returns the token_result partial via HTMX swap.
func (d *DashboardServer) handleSettingsRegenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	newToken, err := d.mgr.RegenerateBootstrapToken(r.Context())
	if err != nil {
		http.Error(w, "failed to regenerate token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := tokenResultData{Token: newToken}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := d.tmpl.ExecuteTemplate(w, "token_result", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}
