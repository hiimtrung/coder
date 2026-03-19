package auth

import "context"

// contextKey is an unexported type so no other package can collide with it.
type contextKey struct{}

// WithClient returns a copy of ctx with the authenticated Client attached.
// Both the HTTP middleware and gRPC interceptor use this to store the caller identity.
func WithClient(ctx context.Context, c *Client) context.Context {
	return context.WithValue(ctx, contextKey{}, c)
}

// ClientFromContext retrieves the authenticated Client stored by WithClient.
// Returns nil if the context carries no client (open mode or un-authenticated path).
func ClientFromContext(ctx context.Context) *Client {
	c, _ := ctx.Value(contextKey{}).(*Client)
	return c
}
