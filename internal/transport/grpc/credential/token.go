// Package credential provides gRPC client-side credentials for coder.
package credential

import "context"

// BearerToken implements credentials.PerRPCCredentials.
// It injects an "authorization: Bearer <token>" metadata entry into every RPC call.
// When Token is empty the credentials are a no-op (open-mode servers).
type BearerToken struct {
	Token string
}

// GetRequestMetadata returns the per-RPC metadata for this credential.
func (b BearerToken) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	if b.Token == "" {
		return nil, nil
	}
	return map[string]string{"authorization": "Bearer " + b.Token}, nil
}

// RequireTransportSecurity returns false so the credential works with plain-text
// (insecure) gRPC connections, which is coder-node's default transport.
func (b BearerToken) RequireTransportSecurity() bool { return false }
