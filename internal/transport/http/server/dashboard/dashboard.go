package dashboard

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	authdomain "github.com/trungtran/coder/internal/domain/auth"
)

//go:embed templates static
var embeddedFS embed.FS

// templateFuncs provides math helpers used by overview charts.
var templateFuncs = template.FuncMap{
	// mulf multiplies two numbers and returns float64.
	"mulf": func(a, b float64) float64 { return a * b },
	// divf divides a by b; returns 0 if b is zero.
	"divf": func(a, b float64) float64 {
		if b == 0 {
			return 0
		}
		return a / b
	},
}

// DashboardServer serves the web dashboard UI.
type DashboardServer struct {
	mgr     authdomain.AuthManager
	version string
	tmpl    *template.Template
}

// NewDashboardServer creates a new DashboardServer.
func NewDashboardServer(mgr authdomain.AuthManager, version string) *DashboardServer {
	tmpl := template.Must(
		template.New("").Funcs(templateFuncs).ParseFS(embeddedFS,
			"templates/*.html",
			"templates/partials/*.html",
		),
	)
	return &DashboardServer{mgr: mgr, version: version, tmpl: tmpl}
}

// RegisterHandlers registers all dashboard routes on the given mux.
func (d *DashboardServer) RegisterHandlers(mux *http.ServeMux) {
	// Static assets — serve embedded static/ subtree under /dashboard/static/
	staticFS, err := fs.Sub(embeddedFS, "static")
	if err != nil {
		panic("dashboard: failed to sub embedded static FS: " + err.Error())
	}
	mux.Handle("/dashboard/static/", http.StripPrefix("/dashboard/static/", http.FileServer(http.FS(staticFS))))

	// Public routes
	mux.HandleFunc("/dashboard/login", d.handleLogin)
	mux.HandleFunc("/dashboard/logout", d.handleLogout)

	// Protected routes — wrapped with dashboard cookie middleware
	auth := dashboardAuthMiddleware(d.mgr)

	mux.Handle("/dashboard", auth(http.HandlerFunc(d.handleRoot)))
	mux.Handle("/dashboard/overview", auth(http.HandlerFunc(d.handleOverview)))
	mux.Handle("/dashboard/clients", auth(http.HandlerFunc(d.handleClients)))
	mux.Handle("/dashboard/clients/", auth(http.HandlerFunc(d.handleClientDetail)))
	mux.Handle("/dashboard/activity", auth(http.HandlerFunc(d.handleActivity)))
	mux.Handle("/dashboard/settings", auth(http.HandlerFunc(d.handleSettings)))
	mux.Handle("/dashboard/settings/regenerate", auth(http.HandlerFunc(d.handleSettingsRegenerate)))
	mux.Handle("/dashboard/stats/chart", auth(http.HandlerFunc(d.handleStatsChart)))
	mux.Handle("/dashboard/stats/commands", auth(http.HandlerFunc(d.handleStatsCommands)))
}
