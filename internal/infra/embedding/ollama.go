package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// OllamaEmbeddingProvider implements domain/memory.EmbeddingProvider using the Ollama API.
type OllamaEmbeddingProvider struct {
	BaseURL string
	Model   string
}

func (p *OllamaEmbeddingProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	url := p.BaseURL
	if url == "" {
		url = "http://localhost:11434"
	}

	// Use endpoint /api/embed (newest)
	apiURL := strings.TrimSuffix(url, "/") + "/api/embed"

	reqBody, _ := json.Marshal(map[string]any{
		"model":    p.Model,
		"input":    text,
		"truncate": true, // truncate gracefully instead of erroring when input exceeds context
	})

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// If /api/embed returns 404, fall back to /api/embeddings (legacy)
	if resp.StatusCode == http.StatusNotFound {
		legacyURL := strings.TrimSuffix(url, "/") + "/api/embeddings"
		legacyBody, _ := json.Marshal(map[string]any{
			"model":    p.Model,
			"prompt":   text,
			"truncate": true,
		})

		req, _ = http.NewRequestWithContext(ctx, "POST", legacyURL, bytes.NewBuffer(legacyBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "" {
			return nil, fmt.Errorf("ollama embedding API error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("ollama embedding API returned error: %s", resp.Status)
	}

	var result struct {
		Embedding  []float32   `json:"embedding"`  // legacy
		Embeddings [][]float32 `json:"embeddings"` // new
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Embeddings) > 0 {
		return result.Embeddings[0], nil
	}
	if len(result.Embedding) > 0 {
		return result.Embedding, nil
	}

	return nil, fmt.Errorf("no embedding returned from ollama (check if model is pulled)")
}
