package httpclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ChatSession represents a persisted conversation session.
type ChatSession struct {
	ID           string    `json:"id"`
	ClientID     string    `json:"client_id"`
	Title        string    `json:"title"`
	MessageCount int       `json:"message_count,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ChatMessage is a single message in a session history.
type ChatMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	TokensIn  int       `json:"tokens_in"`
	TokensOut int       `json:"tokens_out"`
	CreatedAt time.Time `json:"created_at"`
}

// ContextUsed shows which memory/skill entries were injected.
type ContextUsed struct {
	MemoryHits []string `json:"memory_hits"`
	SkillHits  []string `json:"skill_hits"`
}

// ChatResponse is the response from POST /v1/chat.
type ChatResponse struct {
	Reply       string      `json:"reply"`
	SessionID   string      `json:"session_id"`
	ContextUsed ContextUsed `json:"context_used"`
	Model       string      `json:"model"`
	Tokens      struct {
		Prompt     int `json:"prompt"`
		Completion int `json:"completion"`
	} `json:"tokens"`
}

// ChatClient is a typed HTTP client for the /v1/chat and /v1/sessions endpoints.
type ChatClient struct {
	baseURL     string
	client      *http.Client
	accessToken string
}

func NewChatClient(baseURL, accessToken string) *ChatClient {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	return &ChatClient{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		client:      &http.Client{Timeout: 120 * time.Second},
		accessToken: accessToken,
	}
}

func (c *ChatClient) addAuth(req *http.Request) {
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
}

// Chat sends a blocking completion request.
func (c *ChatClient) Chat(ctx context.Context, message, sessionID string, injectMemory, injectSkills bool) (*ChatResponse, error) {
	body := map[string]any{
		"message":    message,
		"session_id": sessionID,
		"context": map[string]any{
			"inject_memory": injectMemory,
			"inject_skills": injectSkills,
		},
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach coder-node at %s\nRun: cd ~/.coder-node && docker compose up -d", c.baseURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
				Action  string `json:"action"`
			} `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errBody)
		if errBody.Error.Code != "" {
			msg := fmt.Sprintf("[%s] %s", errBody.Error.Code, errBody.Error.Message)
			if errBody.Error.Action != "" {
				msg += "\n  → " + errBody.Error.Action
			}
			return nil, fmt.Errorf("%s", msg)
		}
		return nil, fmt.Errorf("coder-node returned %s", resp.Status)
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// ChatStreamDelta is one chunk from the SSE stream.
type ChatStreamDelta struct {
	Delta       string      `json:"delta"`
	Done        bool        `json:"done"`
	SessionID   string      `json:"session_id"`
	ContextUsed ContextUsed `json:"context_used"`
	Error       string      `json:"error"`
	Tokens      struct {
		Prompt     int `json:"prompt"`
		Completion int `json:"completion"`
	} `json:"tokens"`
}

// ChatStream sends a streaming completion. onDelta is called for each token chunk.
// Returns the final metadata (session_id, context_used, tokens) after the stream ends.
func (c *ChatClient) ChatStream(ctx context.Context, message, sessionID string, injectMemory, injectSkills bool, onDelta func(string)) (*ChatStreamDelta, error) {
	body := map[string]any{
		"message":    message,
		"session_id": sessionID,
		"context": map[string]any{
			"inject_memory": injectMemory,
			"inject_skills": injectSkills,
		},
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/stream", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	c.addAuth(req)

	// Use a client without timeout for streaming
	streamClient := &http.Client{}
	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach coder-node at %s\nRun: cd ~/.coder-node && docker compose up -d", c.baseURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coder-node returned %s", resp.Status)
	}

	var final ChatStreamDelta
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")

		var chunk ChatStreamDelta
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if chunk.Error != "" {
			return nil, fmt.Errorf("%s", chunk.Error)
		}
		if chunk.Delta != "" && onDelta != nil {
			onDelta(chunk.Delta)
		}
		if chunk.Done {
			final = chunk
		}
	}

	return &final, nil
}

// ListSessions returns recent chat sessions.
func (c *ChatClient) ListSessions(ctx context.Context) ([]ChatSession, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/v1/sessions", nil)
	if err != nil {
		return nil, err
	}
	c.addAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach coder-node: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Sessions []ChatSession `json:"sessions"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Sessions, nil
}

// GetSession returns a session with its full message history.
func (c *ChatClient) GetSession(ctx context.Context, id string) (*ChatSession, []ChatMessage, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/v1/sessions/"+id, nil)
	if err != nil {
		return nil, nil, err
	}
	c.addAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil, fmt.Errorf("session not found: %s", id)
	}

	var result struct {
		Session  ChatSession   `json:"session"`
		Messages []ChatMessage `json:"messages"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return &result.Session, result.Messages, nil
}

// DeleteSession removes a session.
func (c *ChatClient) DeleteSession(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+"/v1/sessions/"+id, nil)
	if err != nil {
		return err
	}
	c.addAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
