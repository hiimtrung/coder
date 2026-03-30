package grpcclient

import (
	"context"
	"fmt"

	"github.com/trungtran/coder/api/grpc/debugpb"
	debugdomain "github.com/trungtran/coder/internal/domain/debug"
	httpclient "github.com/trungtran/coder/internal/transport/http/client"
	"github.com/trungtran/coder/internal/transport/grpc/credential"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DebugClient is a gRPC implementation of httpclient.DebugClientIface.
type DebugClient struct {
	conn   *grpc.ClientConn
	client debugpb.DebugServiceClient
}

// NewDebugClient dials the coder-node gRPC endpoint and returns a DebugClientIface.
func NewDebugClient(addr, accessToken string) (httpclient.DebugClientIface, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(credential.BearerToken{Token: accessToken}),
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to debug service: %w", err)
	}
	return &DebugClient{
		conn:   conn,
		client: debugpb.NewDebugServiceClient(conn),
	}, nil
}

// Debug analyses an error over gRPC.
func (c *DebugClient) Debug(ctx context.Context, errorMsg, fileContext, diffContext string, injectMemory, injectSkills bool) (*debugdomain.DebugResult, error) {
	req := &debugpb.DebugRequest{
		ErrorMessage: errorMsg,
		FileContext:  fileContext,
		DiffContext:  diffContext,
		Context: &debugpb.DebugContext{
			InjectMemory: injectMemory,
			InjectSkills: injectSkills,
		},
	}

	resp, err := c.client.Debug(ctx, req)
	if err != nil {
		return nil, grpcConnError(err)
	}

	return protoToDebugResult(resp), nil
}

// Close releases the underlying gRPC connection.
func (c *DebugClient) Close() error {
	return c.conn.Close()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func protoToDebugResult(resp *debugpb.DebugResponse) *debugdomain.DebugResult {
	if resp == nil {
		return &debugdomain.DebugResult{}
	}

	result := &debugdomain.DebugResult{
		RootCause:     resp.RootCause,
		Location:      resp.Location,
		Confidence:    resp.Confidence,
		SuggestedFix:  resp.SuggestedFix,
		SimilarIssues: resp.SimilarIssues,
		FollowUp:      resp.FollowUp,
		Model:         resp.Model,
	}
	if resp.ContextUsed != nil {
		result.ContextUsed = debugdomain.ContextUsed{
			MemoryHits: resp.ContextUsed.MemoryHits,
			SkillHits:  resp.ContextUsed.SkillHits,
		}
	}
	return result
}

// Ensure DebugClient implements the interface at compile time.
var _ httpclient.DebugClientIface = (*DebugClient)(nil)
