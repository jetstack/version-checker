package oci

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDiscoverTimestamp(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expected    time.Time
		expectErr   bool
	}{
		{
			name:        "No annotations",
			annotations: map[string]string{},
			expected:    time.Time{},
			expectErr:   false,
		},
		{
			name: "Valid CreatedTimeAnnotation",
			annotations: map[string]string{
				CreatedTimeAnnotation: "2023-03-15T12:34:56Z",
			},
			expected:  time.Date(2023, 3, 15, 12, 34, 56, 0, time.UTC),
			expectErr: false,
		},
		{
			name: "Valid BuildDateAnnotation",
			annotations: map[string]string{
				BuildDateAnnotation: "2023-03-15T12:34:56Z",
			},
			expected:  time.Date(2023, 3, 15, 12, 34, 56, 0, time.UTC),
			expectErr: false,
		},
		{
			name: "Invalid CreatedTimeAnnotation format",
			annotations: map[string]string{
				CreatedTimeAnnotation: "invalid-date",
			},
			expected:  time.Time{},
			expectErr: true,
		},
		{
			name: "Invalid BuildDateAnnotation format",
			annotations: map[string]string{
				BuildDateAnnotation: "invalid-date",
			},
			expected:  time.Time{},
			expectErr: true,
		},
		{
			name: "Both annotations present, prefer CreatedTimeAnnotation",
			annotations: map[string]string{
				CreatedTimeAnnotation: "2023-03-15T12:34:56Z",
				BuildDateAnnotation:   "2023-01-01T00:00:00Z",
			},
			expected:  time.Date(2023, 3, 15, 12, 34, 56, 0, time.UTC),
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := discoverTimestamp(tt.annotations)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
