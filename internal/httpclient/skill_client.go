package httpclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/trungtran/coder/internal/skill"
)

type skillClient struct {
	baseURL string
	client  *http.Client
}

// NewSkillClient creates a new HTTP client for the skill service.
func NewSkillClient(baseURL string) (skill.Client, error) {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &skillClient{
		baseURL: u.String(),
		client:  &http.Client{},
	}, nil
}

func (c *skillClient) IngestSkill(ctx context.Context, name, skillMD string, rules []skill.RuleFile, source, sourceRepo, category string) (*skill.IngestResult, error) {
	ruleEntries := make([]struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}, len(rules))
	for i, r := range rules {
		ruleEntries[i].Path = r.Path
		ruleEntries[i].Content = r.Content
	}

	payload := map[string]interface{}{
		"name":               name,
		"skill_md_content":   skillMD,
		"rules":              ruleEntries,
		"source":             source,
		"source_repo":        sourceRepo,
		"category":           category,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/skill/ingest", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned error: HTTP %d", resp.StatusCode)
	}

	var result struct {
		SkillID     string `json:"skill_id"`
		ChunksTotal int    `json:"chunks_total"`
		ChunksNew   int    `json:"chunks_new"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &skill.IngestResult{
		SkillName:   result.SkillID,
		ChunksTotal: result.ChunksTotal,
		ChunksNew:   result.ChunksNew,
	}, nil
}

func (c *skillClient) SearchSkills(ctx context.Context, query string, limit int) ([]skill.SkillSearchResult, error) {
	payload := map[string]interface{}{
		"query": query,
		"limit": limit,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/skill/search", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned error: HTTP %d", resp.StatusCode)
	}

	var wrapper struct {
		Results []skill.SkillSearchResult `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, err
	}

	return wrapper.Results, nil
}

func (c *skillClient) ListSkills(ctx context.Context, source, category string, limit, offset int) ([]skill.Skill, error) {
	u, _ := url.Parse(c.baseURL + "/v1/skill/list")
	q := u.Query()
	if source != "" {
		q.Set("source", source)
	}
	if category != "" {
		q.Set("category", category)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned error: HTTP %d", resp.StatusCode)
	}

	var wrapper struct {
		Skills []skill.Skill `json:"skills"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, err
	}

	return wrapper.Skills, nil
}

func (c *skillClient) GetSkill(ctx context.Context, name string) (*skill.Skill, []skill.SkillChunk, error) {
	u, _ := url.Parse(c.baseURL + "/v1/skill/info")
	q := u.Query()
	q.Set("name", name)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("server returned error: HTTP %d", resp.StatusCode)
	}

	var wrapper struct {
		Skill  *skill.Skill       `json:"skill"`
		Chunks []skill.SkillChunk `json:"chunks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, nil, err
	}

	return wrapper.Skill, wrapper.Chunks, nil
}

func (c *skillClient) DeleteSkill(ctx context.Context, name string) error {
	u, _ := url.Parse(c.baseURL + "/v1/skill/delete")
	q := u.Query()
	q.Set("name", name)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "DELETE", u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned error: HTTP %d", resp.StatusCode)
	}

	return nil
}

func (c *skillClient) StoreSkillFiles(ctx context.Context, skillName string, files []skill.SkillFile) (int, error) {
	type fileEntry struct {
		RelPath     string `json:"rel_path"`
		ContentType string `json:"content_type"`
		Content     string `json:"content"` // base64-encoded
		SizeBytes   int    `json:"size_bytes"`
	}
	entries := make([]fileEntry, len(files))
	for i, f := range files {
		entries[i] = fileEntry{
			RelPath:     f.RelPath,
			ContentType: f.ContentType,
			Content:     base64.StdEncoding.EncodeToString(f.Content),
			SizeBytes:   f.SizeBytes,
		}
	}

	payload := map[string]interface{}{
		"skill_name": skillName,
		"files":      entries,
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/skill/files", bytes.NewBuffer(data))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned error: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Stored int `json:"stored"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.Stored, nil
}

func (c *skillClient) GetSkillFiles(ctx context.Context, skillName string) ([]skill.SkillFile, error) {
	u, _ := url.Parse(c.baseURL + "/v1/skill/files")
	q := u.Query()
	q.Set("name", skillName)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned error: HTTP %d", resp.StatusCode)
	}

	var wrapper struct {
		Files []struct {
			RelPath     string `json:"rel_path"`
			ContentType string `json:"content_type"`
			Content     string `json:"content"` // base64
			SizeBytes   int    `json:"size_bytes"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, err
	}

	result := make([]skill.SkillFile, 0, len(wrapper.Files))
	for _, f := range wrapper.Files {
		decoded, err := base64.StdEncoding.DecodeString(f.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to decode file %s: %w", f.RelPath, err)
		}
		result = append(result, skill.SkillFile{
			RelPath:     f.RelPath,
			ContentType: f.ContentType,
			Content:     decoded,
			SizeBytes:   f.SizeBytes,
		})
	}
	return result, nil
}

func (c *skillClient) Close() error {
	return nil
}
