package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	debugdomain "github.com/trungtran/coder/internal/domain/debug"
)

// DebugClient is a typed HTTP client for POST /v1/debug.
type DebugClient struct {
	baseURL     string
	client      *http.Client
	accessToken string
}

func NewDebugClient(baseURL, accessToken string) *DebugClient {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	return &DebugClient{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		client:      &http.Client{},
		accessToken: accessToken,
	}
}

func (c *DebugClient) addAuth(req *http.Request) {
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
}

// Debug sends a debug request and returns the structured result.
func (c *DebugClient) Debug(ctx context.Context, errorMsg, fileContext, diffContext string, injectMemory, injectSkills bool) (*debugdomain.DebugResult, error) {
	body := map[string]any{
		"error_message": errorMsg,
		"file_context":  fileContext,
		"diff_context":  diffContext,
		"context": map[string]any{
			"inject_memory": injectMemory,
			"inject_skills": injectSkills,
		},
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/debug", bytes.NewBuffer(data))
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
			} `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errBody)
		if errBody.Error.Code != "" {
			return nil, fmt.Errorf("[%s] %s", errBody.Error.Code, errBody.Error.Message)
		}
		return nil, fmt.Errorf("coder-node returned %s", resp.Status)
	}

	var result debugdomain.DebugResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse debug response: %w", err)
	}
	return &result, nil
}
