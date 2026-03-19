package dashboard

import (
	"net/http"
)

type loginData struct {
	Version   string
	Error     string
	ActiveTab string // "bootstrap" or "access" — remembered on error
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
		d.renderLogin(w, r, "", "bootstrap")
	case http.MethodPost:
		d.processLogin(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (d *DashboardServer) renderLogin(w http.ResponseWriter, _ *http.Request, errMsg, activeTab string) {
	if activeTab == "" {
		activeTab = "bootstrap"
	}
	data := loginData{Version: d.version, Error: errMsg, ActiveTab: activeTab}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := d.tmpl.ExecuteTemplate(w, "login.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (d *DashboardServer) processLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		d.renderLogin(w, r, "Invalid form submission.", "bootstrap")
		return
	}

	loginType := r.FormValue("login_type")
	switch loginType {
	case "access_token":
		d.processAccessTokenLogin(w, r)
	default:
		d.processBootstrapLogin(w, r)
	}
}

// processBootstrapLogin handles login via bootstrap token.
// Registers (or re-registers) the dashboard client and mints a fresh access token.
func (d *DashboardServer) processBootstrapLogin(w http.ResponseWriter, r *http.Request) {
	bootstrapToken := r.FormValue("bootstrap_token")
	if bootstrapToken == "" {
		d.renderLogin(w, r, "Bootstrap token is required.", "bootstrap")
		return
	}

	rawToken, err := d.mgr.RegisterClient(r.Context(), bootstrapToken, "Dashboard Admin", "dashboard@local")
	if err != nil {
		d.renderLogin(w, r, "Invalid bootstrap token.", "bootstrap")
		return
	}

	d.setSessionCookie(w, rawToken)
	http.Redirect(w, r, "/dashboard/overview", http.StatusSeeOther)
}

// processAccessTokenLogin handles login via an existing coder client access token.
// The token is validated; if valid the same token is used as the session cookie value.
func (d *DashboardServer) processAccessTokenLogin(w http.ResponseWriter, r *http.Request) {
	accessToken := r.FormValue("access_token")
	if accessToken == "" {
		d.renderLogin(w, r, "Access token is required.", "access")
		return
	}

	// Validate that the token belongs to a registered client.
	if _, err := d.mgr.ValidateToken(r.Context(), accessToken); err != nil {
		d.renderLogin(w, r, "Invalid or expired access token.", "access")
		return
	}

	d.setSessionCookie(w, accessToken)
	http.Redirect(w, r, "/dashboard/overview", http.StatusSeeOther)
}

func (d *DashboardServer) setSessionCookie(w http.ResponseWriter, rawToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    rawToken,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/dashboard",
		MaxAge:   cookieMaxAge,
	})
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
