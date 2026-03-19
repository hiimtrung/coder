package dashboard

import (
	"net/http"

	authdomain "github.com/trungtran/coder/internal/domain/auth"
)

const cookieName = "coder_dash"
const cookieMaxAge = 86400 * 7 // 7 days

// dashboardAuthMiddleware validates the coder_dash cookie.
// In open mode (IsSecureMode == false) it always passes through.
// In secure mode it validates the token via ValidateToken; on failure it
// redirects the browser to /dashboard/login or returns 401 for HTMX requests.
func dashboardAuthMiddleware(mgr authdomain.AuthManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !mgr.IsSecureMode() {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(cookieName)
			if err != nil {
				// No cookie present
				if r.Header.Get("HX-Request") == "true" {
					http.Error(w, "session expired", http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, "/dashboard/login", http.StatusFound)
				return
			}

			client, err := mgr.ValidateToken(r.Context(), cookie.Value)
			if err != nil {
				http.SetCookie(w, &http.Cookie{
					Name:   cookieName,
					MaxAge: -1,
					Path:   "/dashboard",
				})
				http.Redirect(w, r, "/dashboard/login", http.StatusFound)
				return
			}

			// Renew cookie on each valid request
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    cookie.Value,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				Path:     "/dashboard",
				MaxAge:   cookieMaxAge,
			})

			ctx := authdomain.WithClient(r.Context(), client)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
