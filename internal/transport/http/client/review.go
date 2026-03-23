package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	reviewdomain "github.com/trungtran/coder/internal/domain/review"
)

// ReviewClient is a typed HTTP client for POST /v1/review.
type ReviewClient struct {
	baseURL     string
	client      *http.Client
	accessToken string
}

func NewReviewClient(baseURL, accessToken string) *ReviewClient {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	return &ReviewClient{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		client:      &http.Client{},
		accessToken: accessToken,
	}
}

func (c *ReviewClient) addAuth(req *http.Request) {
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
}

// Review sends a review request and returns the structured result.
func (c *ReviewClient) Review(ctx context.Context, reviewType, content, focus string, injectMemory, injectSkills bool) (*reviewdomain.ReviewResult, error) {
	body := map[string]any{
		"type":    reviewType,
		"content": content,
		"focus":   focus,
		"context": map[string]any{
			"inject_memory": injectMemory,
			"inject_skills": injectSkills,
		},
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/review", bytes.NewBuffer(data))
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

	var result reviewdomain.ReviewResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse review response: %w", err)
	}
	return &result, nil
}
