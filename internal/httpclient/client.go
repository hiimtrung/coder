package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/trungtran/coder/internal/memory"
)

type Client struct {
	baseURL string
	client  *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	// Ensure baseURL has protocol
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		baseURL = "http://" + baseURL
	}

	return &Client{
		baseURL: baseURL,
		client:  &http.Client{},
	}, nil
}

func (c *Client) Store(ctx context.Context, title, content string, memType memory.MemoryType, metadata map[string]interface{}, scope string, tags []string) (string, error) {
	reqBody := map[string]interface{}{
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

func (c *Client) Search(ctx context.Context, query string, scope string, tags []string, memType memory.MemoryType, metaFilters map[string]interface{}, limit int) ([]memory.SearchResult, error) {
	reqBody := map[string]interface{}{
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

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %s", resp.Status)
	}

	var res struct {
		Results []memory.SearchResult `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Results, nil
}

func (c *Client) List(ctx context.Context, limit, offset int) ([]memory.Knowledge, error) {
	u, _ := url.Parse(c.baseURL + "/v1/memory/list")
	q := u.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("offset", fmt.Sprintf("%d", offset))
	u.RawQuery = q.Encode()

	req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %s", resp.Status)
	}

	var res struct {
		Items []memory.Knowledge `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Items, nil
}

func (c *Client) Delete(ctx context.Context, id string) error {
	u, _ := url.Parse(c.baseURL + "/v1/memory/delete")
	q := u.Query()
	q.Set("id", id)
	u.RawQuery = q.Encode()

	req, _ := http.NewRequestWithContext(ctx, "DELETE", u.String(), nil)
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

func (c *Client) Compact(ctx context.Context, threshold float32) (int, error) {
	reqBody := map[string]interface{}{
		"threshold": threshold,
	}

	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/memory/compact", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

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
	reqBody := map[string]interface{}{
		"revector": true,
	}

	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/memory/compact", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

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

func (c *Client) Close() error {
	return nil
}
