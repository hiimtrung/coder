package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	memdomain "github.com/trungtran/coder/internal/domain/memory"
)

// Client is an HTTP memory client that implements memdomain.MemoryManager.
type Client struct {
	baseURL     string
	client      *http.Client
	accessToken string
}

func NewClient(baseURL, accessToken string) (*Client, error) {
	// Ensure baseURL has protocol
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL:     u.String(),
		client:      &http.Client{},
		accessToken: accessToken,
	}, nil
}

// addAuth injects the Authorization header when an access token is configured.
func (c *Client) addAuth(req *http.Request) {
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
}

func (c *Client) Store(ctx context.Context, title, content string, memType memdomain.MemoryType, metadata map[string]any, scope string, tags []string) (string, error) {
	reqBody := map[string]any{
		"title":    title,
		"content":  content,
		"type":     string(memType),
		"metadata": metadata,
		"scope":    scope,
		"tags":     tags,
	}

	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/memory/store", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status: %s", resp.Status)
	}

	var res struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.ID, nil
}

func (c *Client) Search(ctx context.Context, query string, scope string, tags []string, memType memdomain.MemoryType, metaFilters map[string]any, limit int) ([]memdomain.SearchResult, error) {
	reqBody := map[string]any{
		"query":        query,
		"scope":        scope,
		"tags":         tags,
		"type":         string(memType),
		"meta_filters": metaFilters,
		"limit":        limit,
	}

	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/memory/search", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %s", resp.Status)
	}

	var res struct {
		Results []memdomain.SearchResult `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	for i := range res.Results {
		memdomain.HydrateKnowledgeLifecycle(&res.Results[i].Knowledge)
	}
	return res.Results, nil
}

func (c *Client) Recall(ctx context.Context, opts memdomain.RecallOptions) (memdomain.RecallResult, error) {
	reqBody := map[string]any{
		"task":         opts.Task,
		"current":      opts.Current,
		"trigger":      opts.Trigger,
		"budget":       opts.Budget,
		"limit":        opts.Limit,
		"scope":        opts.Scope,
		"tags":         opts.Tags,
		"type":         string(opts.Type),
		"meta_filters": opts.MetaFilters,
	}

	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/memory/recall", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return memdomain.RecallResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return memdomain.RecallResult{}, fmt.Errorf("server returned status: %s", resp.Status)
	}

	var res memdomain.RecallResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return memdomain.RecallResult{}, err
	}
	for i := range res.Memories {
		memdomain.HydrateKnowledgeLifecycle(&res.Memories[i].Result.Knowledge)
	}
	return res, nil
}

func (c *Client) List(ctx context.Context, limit, offset int) ([]memdomain.Knowledge, error) {
	u, _ := url.Parse(c.baseURL + "/v1/memory/list")
	q := u.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("offset", fmt.Sprintf("%d", offset))
	u.RawQuery = q.Encode()

	req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	c.addAuth(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %s", resp.Status)
	}

	var res struct {
		Items []memdomain.Knowledge `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	for i := range res.Items {
		memdomain.HydrateKnowledgeLifecycle(&res.Items[i])
	}
	return res.Items, nil
}

func (c *Client) Delete(ctx context.Context, id string) error {
	u, _ := url.Parse(c.baseURL + "/v1/memory/delete")
	q := u.Query()
	q.Set("id", id)
	u.RawQuery = q.Encode()

	req, _ := http.NewRequestWithContext(ctx, "DELETE", u.String(), nil)
	c.addAuth(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}
	return nil
}

func (c *Client) Verify(ctx context.Context, id string, opts memdomain.VerifyOptions) (int, error) {
	reqBody := map[string]any{
		"id":          id,
		"verified_at": opts.VerifiedAt,
		"verified_by": opts.VerifiedBy,
		"confidence":  opts.Confidence,
		"source_ref":  opts.SourceRef,
	}

	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/memory/verify", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned status: %s", resp.Status)
	}

	var res struct {
		UpdatedCount int `json:"updated_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}
	return res.UpdatedCount, nil
}

func (c *Client) Supersede(ctx context.Context, id string, replacementID string) (int, error) {
	reqBody := map[string]any{
		"id":             id,
		"replacement_id": replacementID,
	}

	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/memory/supersede", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned status: %s", resp.Status)
	}

	var res struct {
		UpdatedCount int `json:"updated_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}
	return res.UpdatedCount, nil
}

func (c *Client) Audit(ctx context.Context, opts memdomain.AuditOptions) (memdomain.AuditReport, error) {
	data, _ := json.Marshal(opts)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/memory/audit", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return memdomain.AuditReport{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return memdomain.AuditReport{}, fmt.Errorf("server returned status: %s", resp.Status)
	}

	var res struct {
		Report memdomain.AuditReport `json:"report"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return memdomain.AuditReport{}, err
	}
	return res.Report, nil
}

func (c *Client) Compact(ctx context.Context, threshold float32) (int, error) {
	reqBody := map[string]any{
		"threshold": threshold,
	}

	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/memory/compact", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned status: %s", resp.Status)
	}

	var res struct {
		RemovedCount int `json:"removed_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}
	return res.RemovedCount, nil
}

func (c *Client) Revector(ctx context.Context) error {
	reqBody := map[string]any{
		"revector": true,
	}

	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/memory/compact", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}
	return nil
}

// LogActivity sends a command activity record to coder-node.
func (c *Client) LogActivity(ctx context.Context, command, repo, branch string) error {
	payload := map[string]string{
		"command": command,
		"repo":    repo,
		"branch":  branch,
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/auth/log-activity", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *Client) Close() error {
	return nil
}
