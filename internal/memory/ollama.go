package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type OllamaEmbeddingProvider struct {
	BaseURL string
	Model   string
}

func (p *OllamaEmbeddingProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	url := p.BaseURL
	if url == "" {
		url = "http://localhost:11434"
	}

	// Đảm bảo URL kết thúc đúng API endpoint của Ollama
	if !strings.HasSuffix(url, "/api/embeddings") {
		url = strings.TrimSuffix(url, "/") + "/api/embeddings"
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":  p.Model,
		"prompt": text,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embedding API returned error: %s", resp.Status)
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("no embedding returned from ollama")
	}

	return result.Embedding, nil
}
