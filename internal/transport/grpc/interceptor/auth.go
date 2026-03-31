// Package interceptor provides gRPC server-side interceptors for coder-node.
package interceptor

import (
	"context"
	"strings"

	authdomain "github.com/trungtran/coder/internal/domain/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryAuth returns a gRPC unary interceptor that validates Bearer tokens.
// In open mode (IsSecureMode == false) it is a transparent no-op.
func UnaryAuth(mgr authdomain.AuthManager) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if !mgr.IsSecureMode() || isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}
		ctx, err := validateToken(ctx, mgr)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamAuth returns a gRPC stream interceptor that validates Bearer tokens.
// In open mode it is a transparent no-op.
func StreamAuth(mgr authdomain.AuthManager) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if !mgr.IsSecureMode() || isPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}
		ctx, err := validateToken(ss.Context(), mgr)
		if err != nil {
			return err
		}
		return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
	}
}

// validateToken extracts and validates the Bearer token from incoming gRPC metadata.
// On success it attaches the authenticated *Client to the returned context.
func validateToken(ctx context.Context, mgr authdomain.AuthManager) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, status.Error(codes.Unauthenticated, "AUTH_TOKEN_MISSING: no metadata in request")
	}

	vals := md.Get("authorization")
	if len(vals) == 0 {
		return ctx, status.Error(codes.Unauthenticated, "AUTH_TOKEN_MISSING: authorization header required")
	}

	raw, found := strings.CutPrefix(vals[0], "Bearer ")
	if !found || raw == "" {
		return ctx, status.Error(codes.Unauthenticated, "AUTH_TOKEN_MISSING: authorization must be 'Bearer <token>'")
	}

	client, err := mgr.ValidateToken(ctx, raw)
	if err != nil {
		return ctx, status.Error(codes.Unauthenticated, "AUTH_TOKEN_INVALID: invalid or expired access token")
	}

	return authdomain.WithClient(ctx, client), nil
}

// wrappedStream replaces the embedded context so interceptors can propagate
// the enriched context (with Client) down to the stream handler.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context { return w.ctx }

func isPublicMethod(fullMethod string) bool {
	switch fullMethod {
	case "/auth.AuthService/RegisterClient":
		return true
	default:
		return false
	}
}
