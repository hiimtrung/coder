package httpmiddleware

import (
	"net/http"
	"strings"

	authdomain "github.com/trungtran/coder/internal/domain/auth"
)

// Auth returns an HTTP middleware that enforces Bearer token authentication
// when the AuthManager is in secure mode. Unauthenticated requests get 401.
// Public paths: /v1/auth/register-client, /health
func Auth(mgr authdomain.AuthManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !mgr.IsSecureMode() || isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":{"code":"AUTH_TOKEN_MISSING","message":"Authorization header required"}}`, http.StatusUnauthorized)
				return
			}

			rawToken := strings.TrimPrefix(authHeader, "Bearer ")
			client, err := mgr.ValidateToken(r.Context(), rawToken)
			if err != nil {
				http.Error(w, `{"error":{"code":"AUTH_TOKEN_INVALID","message":"Invalid or expired access token"}}`, http.StatusUnauthorized)
				return
			}

			ctx := authdomain.WithClient(r.Context(), client)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClientFromContext retrieves the authenticated client from an HTTP request context.
// Delegates to the domain-level helper so callers need not import the domain package.
func ClientFromContext(r *http.Request) *authdomain.Client {
	return authdomain.ClientFromContext(r.Context())
}

func isPublicPath(path string) bool {
	switch path {
	case "/v1/auth/register-client",
		"/v1/auth/bootstrap/status",
		"/health":
		return true
	}
	return false
}
