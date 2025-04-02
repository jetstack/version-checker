package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNoVersionFound(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		expectRes bool
	}{
		{
			name:      "error is of type ErrorVersionNotFound",
			err:       NewVersionErrorNotFound("version not found"),
			expectRes: true,
		},
		{
			name:      "error is not of type ErrorVersionNotFound",
			err:       errors.New("some other error"),
			expectRes: false,
		},
		{
			name:      "error is nil",
			err:       nil,
			expectRes: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsNoVersionFound(test.err)
			assert.Equal(t, result, test.expectRes)
		})
	}
}
