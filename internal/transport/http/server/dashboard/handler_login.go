package dashboard

import (
	"net/http"
)

type loginData struct {
	Version string
	Error   string
}

// handleLogin serves GET /dashboard/login and handles POST /dashboard/login.
func (d *DashboardServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	// In open mode skip authentication entirely — go straight to overview.
	if !d.mgr.IsSecureMode() {
		http.Redirect(w, r, "/dashboard/overview", http.StatusFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		d.renderLogin(w, r, "")
	case http.MethodPost:
		d.processLogin(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (d *DashboardServer) renderLogin(w http.ResponseWriter, r *http.Request, errMsg string) {
	data := loginData{Version: d.version, Error: errMsg}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := d.tmpl.ExecuteTemplate(w, "login.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (d *DashboardServer) processLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		d.renderLogin(w, r, "Invalid form submission.")
		return
	}

	bootstrapToken := r.FormValue("bootstrap_token")
	if bootstrapToken == "" {
		d.renderLogin(w, r, "Bootstrap token is required.")
		return
	}

	// Register (or re-register) the dashboard client using the bootstrap token.
	rawToken, err := d.mgr.RegisterClient(r.Context(), bootstrapToken, "Dashboard Admin", "dashboard@local")
	if err != nil {
		d.renderLogin(w, r, "Invalid bootstrap token.")
		return
	}

	// Set session cookie with the raw access token.
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    rawToken,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/dashboard",
		MaxAge:   cookieMaxAge,
	})
	http.Redirect(w, r, "/dashboard/overview", http.StatusSeeOther)
}

// handleLogout clears the session cookie and redirects to login.
func (d *DashboardServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   cookieName,
		MaxAge: -1,
		Path:   "/dashboard",
	})
	http.Redirect(w, r, "/dashboard/login", http.StatusFound)
}

// handleRoot redirects /dashboard to /dashboard/overview.
func (d *DashboardServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/dashboard/overview", http.StatusFound)
}
