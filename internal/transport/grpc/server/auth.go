package grpcserver

import (
	"context"
	"fmt"
	"strings"

	authpb "github.com/trungtran/coder/api/grpc/authpb/auth"
	authdomain "github.com/trungtran/coder/internal/domain/auth"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthServer struct {
	authpb.UnimplementedAuthServiceServer
	mgr authdomain.AuthManager
}

func NewAuthServer(mgr authdomain.AuthManager) *AuthServer {
	return &AuthServer{mgr: mgr}
}

func (s *AuthServer) RegisterClient(ctx context.Context, req *authpb.RegisterClientRequest) (*authpb.RegisterClientResponse, error) {
	rawToken, err := s.mgr.RegisterClient(ctx, req.BootstrapToken, req.GitName, req.GitEmail)
	if err != nil {
		return nil, err
	}
	return &authpb.RegisterClientResponse{
		AccessToken: rawToken,
		Message:     fmt.Sprintf("Client registered as %s. Store this token safely — it will not be shown again.", req.GitEmail),
	}, nil
}

func (s *AuthServer) Me(ctx context.Context, _ *authpb.MeRequest) (*authpb.MeResponse, error) {
	client := authdomain.ClientFromContext(ctx)
	if client == nil {
		return &authpb.MeResponse{
			Id:         "anonymous",
			GitName:    "anonymous",
			SecureMode: s.mgr.IsSecureMode(),
		}, nil
	}

	return &authpb.MeResponse{
		Id:         client.ID,
		GitName:    client.GitName,
		GitEmail:   client.GitEmail,
		CreatedAt:  timestamppb.New(client.CreatedAt),
		LastSeenAt: timestamppb.New(client.LastSeenAt),
		SecureMode: s.mgr.IsSecureMode(),
	}, nil
}

func (s *AuthServer) RotateToken(ctx context.Context, _ *authpb.RotateTokenRequest) (*authpb.RotateTokenResponse, error) {
	client := authdomain.ClientFromContext(ctx)
	if client == nil || client.ID == "anonymous" {
		return nil, fmt.Errorf("authentication required")
	}

	newToken, err := s.mgr.RotateToken(ctx, client.ID)
	if err != nil {
		return nil, err
	}

	return &authpb.RotateTokenResponse{
		AccessToken: newToken,
		Message:     "Token rotated successfully. Update your config with the new token — the old token is now invalid.",
	}, nil
}

func (s *AuthServer) LogActivity(ctx context.Context, req *authpb.LogActivityRequest) (*authpb.LogActivityResponse, error) {
	rawToken := bearerTokenFromMetadata(ctx)
	if err := s.mgr.LogActivity(ctx, rawToken, req.Command, req.Repo, req.Branch); err != nil {
		return &authpb.LogActivityResponse{Success: false}, nil
	}
	return &authpb.LogActivityResponse{Success: true}, nil
}

func bearerTokenFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get("authorization")
	if len(vals) == 0 {
		return ""
	}
	raw, found := strings.CutPrefix(vals[0], "Bearer ")
	if !found {
		return ""
	}
	return raw
}
