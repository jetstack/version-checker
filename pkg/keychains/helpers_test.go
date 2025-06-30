package keychains

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestPullSecrets(t *testing.T) {
	tests := []struct {
		name     string
		secrets  []v1.LocalObjectReference
		expected []string
	}{
		{
			name:     "No secrets",
			secrets:  []v1.LocalObjectReference{},
			expected: []string{},
		},
		{
			name: "Single secret",
			secrets: []v1.LocalObjectReference{
				{Name: "secret-a"},
			},
			expected: []string{"secret-a"},
		},
		{
			name: "Multiple secrets with no duplicates",
			secrets: []v1.LocalObjectReference{
				{Name: "secret-b"},
				{Name: "secret-a"},
				{Name: "secret-c"},
			},
			expected: []string{"secret-a", "secret-b", "secret-c"},
		},
		{
			name: "Multiple secrets with duplicates",
			secrets: []v1.LocalObjectReference{
				{Name: "secret-a"},
				{Name: "secret-b"},
				{Name: "secret-a"},
				{Name: "secret-c"},
				{Name: "secret-b"},
			},
			expected: []string{"secret-a", "secret-b", "secret-c"},
		},
		{
			name: "Secrets already sorted",
			secrets: []v1.LocalObjectReference{
				{Name: "secret-a"},
				{Name: "secret-b"},
				{Name: "secret-c"},
			},
			expected: []string{"secret-a", "secret-b", "secret-c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pullSecrets(tt.secrets)
			assert.Equal(t, tt.expected, result)
		})
	}
}
