package memory

import (
	"math"
)

// CosineSimilarity calculates the cosine similarity between two vectors
func CosineSimilarity(v1, v2 []float32) float32 {
	if len(v1) != len(v2) || len(v1) == 0 {
		return 0
	}

	var dotProduct, normV1, normV2 float64
	for i := range v1 {
		dotProduct += float64(v1[i] * v2[i])
		normV1 += float64(v1[i] * v1[i])
		normV2 += float64(v2[i] * v2[i])
	}

	if normV1 == 0 || normV2 == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normV1) * math.Sqrt(normV2)))
}

// RankResults sorts knowledge items by similarity score
func RankResults(queryVector []float32, items []Knowledge, limit int) []SearchResult {
	results := make([]SearchResult, len(items))
	for i, item := range items {
		score := CosineSimilarity(queryVector, item.Vector)
		results[i] = SearchResult{
			Knowledge: item,
			Score:     score,
		}
	}

	// Simple selection sort or use sort.Slice for better performance if needed
	// For CLI scale, this is fine
	for i := 0; i < len(results); i++ {
		maxIdx := i
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[maxIdx].Score {
				maxIdx = j
			}
		}
		results[i], results[maxIdx] = results[maxIdx], results[i]
	}

	if len(results) > limit {
		return results[:limit]
	}
	return results
}
