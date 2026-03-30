package grpcclient

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/trungtran/coder/api/grpc/chatpb"
	httpclient "github.com/trungtran/coder/internal/transport/http/client"
	"github.com/trungtran/coder/internal/transport/grpc/credential"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// ChatClient is a gRPC implementation of httpclient.ChatClientIface.
type ChatClient struct {
	conn   *grpc.ClientConn
	client chatpb.ChatServiceClient
}

// NewChatClient dials the coder-node gRPC endpoint and returns a ChatClientIface.
func NewChatClient(addr, accessToken string) (httpclient.ChatClientIface, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(credential.BearerToken{Token: accessToken}),
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to chat service: %w", err)
	}
	return &ChatClient{
		conn:   conn,
		client: chatpb.NewChatServiceClient(conn),
	}, nil
}

// ChatStream implements streaming completion over gRPC server-side streaming.
func (c *ChatClient) ChatStream(ctx context.Context, message, sessionID string, injectMemory, injectSkills bool, onDelta func(string)) (*httpclient.ChatStreamDelta, error) {
	req := &chatpb.ChatRequest{
		Message:   message,
		SessionId: sessionID,
		Context: &chatpb.ChatContext{
			InjectMemory: injectMemory,
			InjectSkills: injectSkills,
		},
	}

	stream, err := c.client.ChatStream(ctx, req)
	if err != nil {
		return nil, grpcConnError(err)
	}

	var final httpclient.ChatStreamDelta
	for {
		frame, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, grpcConnError(err)
		}
		if frame.Error != "" {
			return nil, fmt.Errorf("%s", frame.Error)
		}
		if frame.Delta != "" && onDelta != nil {
			onDelta(frame.Delta)
		}
		if frame.Done {
			final = httpclient.ChatStreamDelta{
				Done:      true,
				SessionID: frame.SessionId,
			}
			if frame.ContextUsed != nil {
				final.ContextUsed = httpclient.ContextUsed{
					MemoryHits: frame.ContextUsed.MemoryHits,
					SkillHits:  frame.ContextUsed.SkillHits,
				}
			}
			if frame.Tokens != nil {
				final.Tokens.Prompt = int(frame.Tokens.Prompt)
				final.Tokens.Completion = int(frame.Tokens.Completion)
			}
		}
	}
	return &final, nil
}

// Chat implements blocking completion.
func (c *ChatClient) Chat(ctx context.Context, message, sessionID string, injectMemory, injectSkills bool) (*httpclient.ChatResponse, error) {
	req := &chatpb.ChatRequest{
		Message:   message,
		SessionId: sessionID,
		Context: &chatpb.ChatContext{
			InjectMemory: injectMemory,
			InjectSkills: injectSkills,
		},
	}

	resp, err := c.client.Chat(ctx, req)
	if err != nil {
		return nil, grpcConnError(err)
	}

	result := &httpclient.ChatResponse{
		Reply:     resp.Reply,
		SessionID: resp.SessionId,
		Model:     resp.Model,
	}
	if resp.ContextUsed != nil {
		result.ContextUsed = httpclient.ContextUsed{
			MemoryHits: resp.ContextUsed.MemoryHits,
			SkillHits:  resp.ContextUsed.SkillHits,
		}
	}
	if resp.Tokens != nil {
		result.Tokens.Prompt = int(resp.Tokens.Prompt)
		result.Tokens.Completion = int(resp.Tokens.Completion)
	}
	return result, nil
}

// ListSessions returns recent sessions.
func (c *ChatClient) ListSessions(ctx context.Context) ([]httpclient.ChatSession, error) {
	resp, err := c.client.ListSessions(ctx, &chatpb.ListSessionsRequest{Limit: 50})
	if err != nil {
		return nil, grpcConnError(err)
	}

	var sessions []httpclient.ChatSession
	for _, s := range resp.Sessions {
		sessions = append(sessions, protoToSession(s))
	}
	return sessions, nil
}

// GetSession returns a session with its full message history.
func (c *ChatClient) GetSession(ctx context.Context, id string) (*httpclient.ChatSession, []httpclient.ChatMessage, error) {
	resp, err := c.client.GetSession(ctx, &chatpb.GetSessionRequest{Id: id})
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.NotFound {
			return nil, nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, nil, grpcConnError(err)
	}

	sess := protoToSession(resp.Session)
	var msgs []httpclient.ChatMessage
	for _, m := range resp.Messages {
		msgs = append(msgs, httpclient.ChatMessage{
			ID:        m.Id,
			SessionID: m.SessionId,
			Role:      m.Role,
			Content:   m.Content,
			TokensIn:  int(m.TokensIn),
			TokensOut: int(m.TokensOut),
			CreatedAt: m.CreatedAt.AsTime(),
		})
	}
	return &sess, msgs, nil
}

// DeleteSession removes a session.
func (c *ChatClient) DeleteSession(ctx context.Context, id string) error {
	_, err := c.client.DeleteSession(ctx, &chatpb.DeleteSessionRequest{Id: id})
	return grpcConnError(err)
}

// Close releases the underlying gRPC connection.
func (c *ChatClient) Close() error {
	return c.conn.Close()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func protoToSession(s *chatpb.Session) httpclient.ChatSession {
	if s == nil {
		return httpclient.ChatSession{}
	}
	sess := httpclient.ChatSession{
		ID:       s.Id,
		ClientID: s.ClientId,
		Title:    s.Title,
	}
	if s.CreatedAt != nil {
		sess.CreatedAt = s.CreatedAt.AsTime()
	}
	if s.UpdatedAt != nil {
		sess.UpdatedAt = s.UpdatedAt.AsTime()
	}
	return sess
}

// grpcConnError converts gRPC errors to user-friendly messages.
func grpcConnError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if ok && (st.Code() == codes.Unavailable || st.Code() == codes.DeadlineExceeded) {
		return fmt.Errorf("cannot reach coder-node at gRPC endpoint\nRun: cd ~/.coder-node && docker compose up -d")
	}
	return err
}

// Ensure ChatClient implements the interface at compile time.
var _ httpclient.ChatClientIface = (*ChatClient)(nil)

// dummy time usage to satisfy import
var _ = time.Time{}
