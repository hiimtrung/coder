package memory

import (
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		v1       []float32
		v2       []float32
		expected float32
	}{
		{
			name:     "identical vectors",
			v1:       []float32{1, 0, 0},
			v2:       []float32{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			v1:       []float32{1, 0, 0},
			v2:       []float32{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			v1:       []float32{1, 0, 0},
			v2:       []float32{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "different magnitude",
			v1:       []float32{2, 0, 0},
			v2:       []float32{1, 0, 0},
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CosineSimilarity(tt.v1, tt.v2)
			if got != tt.expected {
				t.Errorf("CosineSimilarity() = %v, want %v", got, tt.expected)
			}
		})
	}
}
