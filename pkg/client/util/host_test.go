package util

import (
	"strings"
	"testing"
)

func TestValidHost(t *testing.T) {
	for _, ok := range []string{"ghcr.io", "registry.example.com:5000", "localhost:5000", "[::1]:5000", "10.0.0.1"} {
		if err := ValidHost(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{
		"", "evil.com#.registry.example.com", "evil.com?.registry.example.com",
		"x@169.254.169.254", "127.0.0.1?x", "a b", "a\\b", "a/b", "-a.com", "a..com", "host:99999999",
		strings.Repeat("a", 300),
	} {
		if ValidHost(bad) == nil {
			t.Errorf("%q should be invalid", bad)
		}
	}
}
