package memory

import (
	"math"
	"strings"
)

// Chunker splits text into overlapping chunks
type Chunker struct {
	ChunkSize    int
	ChunkOverlap int
}

// NewChunker creates a new Chunker with default values
func NewChunker(size, overlap int) *Chunker {
	if size <= 0 {
		size = 1000 // characters
	}
	if overlap < 0 || overlap >= size {
		overlap = 100
	}
	return &Chunker{
		ChunkSize:    size,
		ChunkOverlap: overlap,
	}
}

// Chunk splits text into segments
func (c *Chunker) Chunk(text string) []string {
	text = strings.TrimSpace(text)
	if len(text) <= c.ChunkSize {
		return []string{text}
	}

	var chunks []string
	start := 0
	for start < len(text) {
		end := start + c.ChunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to find a good breaking point (newline or space) if not at the end
		if end < len(text) {
			breakPoint := -1
			// Look back for a newline
			if idx := strings.LastIndexAny(text[start:end], "\n\r"); idx > c.ChunkSize/2 {
				breakPoint = start + idx + 1
			} else if idx := strings.LastIndexAny(text[start:end], " .?!"); idx > c.ChunkSize/2 {
				// Look back for a sentence or word break
				breakPoint = start + idx + 1
			}

			if breakPoint != -1 {
				end = breakPoint
			}
		}

		chunk := strings.TrimSpace(text[start:end])
		if len(chunk) > 0 {
			chunks = append(chunks, chunk)
		}

		if end == len(text) {
			break
		}

		start = end - c.ChunkOverlap
		if start < 0 {
			start = 0
		}

		// Avoid infinite loop if no progress made
		if start >= end {
			start = end
		}

		// Ensure we don't exceed the string length
		if start >= len(text) {
			break
		}
	}

	return chunks
}

// CalculateSimilarityThreshold returns a suggestive threshold for compaction
func CalculateSimilarityThreshold(count int) float32 {
	if count < 100 {
		return 0.95
	}
	return float32(0.85 + 0.1*(1.0/math.Log10(float64(count))))
}
