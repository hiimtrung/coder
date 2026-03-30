package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// OpenAIEmbeddingProvider implements domain/memory.EmbeddingProvider using the OpenAI embeddings API.
// Compatible with OpenAI, Azure OpenAI, and any OpenAI-compatible embedding service.
type OpenAIEmbeddingProvider struct {
	APIKey  string
	Model   string
	BaseURL string
	// Dimensions sets the output vector size. Only supported by text-embedding-3-* models.
	// Default 0 means use the model's native dimensions.
	// Set to 1024 to match the existing pgvector schema (vector(1024)).
	Dimensions int
}

func (p *OpenAIEmbeddingProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	url := p.BaseURL
	if url == "" {
		url = "https://api.openai.com/v1/embeddings"
	} else if !strings.HasSuffix(url, "/embeddings") {
		url = strings.TrimSuffix(url, "/") + "/embeddings"
	}

	reqData := map[string]any{
		"input": text,
		"model": p.Model,
	}
	if p.Dimensions > 0 {
		reqData["dimensions"] = p.Dimensions
	}

	reqBody, _ := json.Marshal(reqData)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("INF_EMBEDDING_UNREACHABLE: cannot reach embedding API at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error.Message != "" {
			return nil, fmt.Errorf("INF_EMBEDDING_ERROR: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("INF_EMBEDDING_ERROR: API returned %s", resp.Status)
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("INF_EMBEDDING_PARSE: failed to parse embedding response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("INF_EMBEDDING_ERROR: no embedding data returned")
	}

	return result.Data[0].Embedding, nil
}
