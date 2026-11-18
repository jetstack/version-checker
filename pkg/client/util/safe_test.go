package util

import "testing"

func TestSafePath(t *testing.T) {
	for _, bad := range []string{"a/../b", "a//b", "../x", "."} {
		if _, err := SafePath(bad); err == nil {
			t.Errorf("SafePath(%q) should fail", bad)
		}
	}
	if got, _ := SafePath("a/b?c"); got != "a/b%3Fc" {
		t.Errorf("got %q", got)
	}
}
