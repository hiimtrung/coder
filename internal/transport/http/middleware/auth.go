package httpmiddleware

import (
	"context"
	"net/http"
	"strings"

	authdomain "github.com/trungtran/coder/internal/domain/auth"
)

type contextKey string

const ClientContextKey contextKey = "auth_client"

// Auth returns an HTTP middleware that enforces Bearer token authentication
// when the AuthManager is in secure mode. Unauthenticated requests get 401.
// The following paths are always public: /v1/auth/register-client, /health.
func Auth(mgr authdomain.AuthManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Public endpoints — always allowed
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

			// Attach client to request context
			ctx := context.WithValue(r.Context(), ClientContextKey, client)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClientFromContext retrieves the authenticated client from the request context.
func ClientFromContext(ctx context.Context) *authdomain.Client {
	c, _ := ctx.Value(ClientContextKey).(*authdomain.Client)
	return c
}

func isPublicPath(path string) bool {
	switch path {
	case "/v1/auth/register-client", "/health":
		return true
	}
	return false
}
