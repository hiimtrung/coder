package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	chatdomain "github.com/trungtran/coder/internal/domain/chat"
)

// OllamaProvider implements chatdomain.LLMProvider using the Ollama /api/chat API.
type OllamaProvider struct {
	BaseURL string
	Model   string
}

func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "qwen3.5:0.8b"
	}
	return &OllamaProvider{BaseURL: strings.TrimSuffix(baseURL, "/"), Model: model}
}

// ollamaMessage is the wire format for Ollama /api/chat.
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaChatResponse struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done             bool `json:"done"`
	PromptEvalCount  int  `json:"prompt_eval_count"`
	EvalCount        int  `json:"eval_count"`
}

// Chat performs a blocking completion.
func (p *OllamaProvider) Chat(ctx context.Context, model string, messages []chatdomain.LLMMessage) (*chatdomain.LLMResponse, error) {
	if model == "" {
		model = p.Model
	}

	msgs := make([]ollamaMessage, len(messages))
	for i, m := range messages {
		msgs[i] = ollamaMessage{Role: m.Role, Content: m.Content}
	}

	body, _ := json.Marshal(ollamaChatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   false,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", p.BaseURL+"/api/chat", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("INF_LLM_UNREACHABLE: cannot reach Ollama at %s — is it running? (%w)", p.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct{ Error string `json:"error"` }
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "" {
			return nil, fmt.Errorf("INF_LLM_ERROR: %s", errResp.Error)
		}
		return nil, fmt.Errorf("INF_LLM_ERROR: Ollama returned %s", resp.Status)
	}

	var result ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("INF_LLM_PARSE: failed to parse Ollama response: %w", err)
	}

	return &chatdomain.LLMResponse{
		Content:   result.Message.Content,
		TokensIn:  result.PromptEvalCount,
		TokensOut: result.EvalCount,
	}, nil
}

// ChatStream streams response deltas to onDelta, then returns the full response.
func (p *OllamaProvider) ChatStream(ctx context.Context, model string, messages []chatdomain.LLMMessage, onDelta func(string)) (*chatdomain.LLMResponse, error) {
	if model == "" {
		model = p.Model
	}

	msgs := make([]ollamaMessage, len(messages))
	for i, m := range messages {
		msgs[i] = ollamaMessage{Role: m.Role, Content: m.Content}
	}

	body, _ := json.Marshal(ollamaChatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   true,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", p.BaseURL+"/api/chat", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("INF_LLM_UNREACHABLE: cannot reach Ollama at %s — is it running? (%w)", p.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct{ Error string `json:"error"` }
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "" {
			return nil, fmt.Errorf("INF_LLM_ERROR: %s", errResp.Error)
		}
		return nil, fmt.Errorf("INF_LLM_ERROR: Ollama returned %s", resp.Status)
	}

	var fullContent strings.Builder
	var tokensIn, tokensOut int

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var chunk ollamaChatResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		if chunk.Message.Content != "" {
			fullContent.WriteString(chunk.Message.Content)
			if onDelta != nil {
				onDelta(chunk.Message.Content)
			}
		}
		if chunk.Done {
			tokensIn = chunk.PromptEvalCount
			tokensOut = chunk.EvalCount
		}
	}

	return &chatdomain.LLMResponse{
		Content:   fullContent.String(),
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
	}, nil
}
