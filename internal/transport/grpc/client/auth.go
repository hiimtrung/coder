package grpcclient

import (
	"context"

	authpb "github.com/trungtran/coder/api/grpc/authpb/auth"
	"github.com/trungtran/coder/internal/transport/grpc/credential"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthClient struct {
	conn *grpc.ClientConn
	c    authpb.AuthServiceClient
}

func NewAuthClient(target, accessToken string) (*AuthClient, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(credential.BearerToken{Token: accessToken}),
	}
	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, err
	}
	return &AuthClient{conn: conn, c: authpb.NewAuthServiceClient(conn)}, nil
}

func (c *AuthClient) RegisterClient(ctx context.Context, bootstrapToken, gitName, gitEmail string) (string, string, error) {
	res, err := c.c.RegisterClient(ctx, &authpb.RegisterClientRequest{
		BootstrapToken: bootstrapToken,
		GitName:        gitName,
		GitEmail:       gitEmail,
	})
	if err != nil {
		return "", "", err
	}
	return res.AccessToken, res.Message, nil
}

func (c *AuthClient) Me(ctx context.Context) (*authpb.MeResponse, error) {
	return c.c.Me(ctx, &authpb.MeRequest{})
}

func (c *AuthClient) RotateToken(ctx context.Context) (string, string, error) {
	res, err := c.c.RotateToken(ctx, &authpb.RotateTokenRequest{})
	if err != nil {
		return "", "", err
	}
	return res.AccessToken, res.Message, nil
}

func (c *AuthClient) LogActivity(ctx context.Context, command, repo, branch string) error {
	_, err := c.c.LogActivity(ctx, &authpb.LogActivityRequest{
		Command: command,
		Repo:    repo,
		Branch:  branch,
	})
	return err
}

func (c *AuthClient) Close() error { return c.conn.Close() }
