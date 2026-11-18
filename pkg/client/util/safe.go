package util

import (
	"fmt"
	"io"
	"net/url"
	"strings"
)

// MaxResponseBytes caps how much of a registry response is read into memory.
const MaxResponseBytes = 16 << 20

// ReadAll reads r, failing if it holds more than MaxResponseBytes.
func ReadAll(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxResponseBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", MaxResponseBytes)
	}
	return b, nil
}

// Truncate shortens b for logs and error strings, so remote bodies are not reflected whole.
func Truncate(b []byte) string {
	const max = 256
	if len(b) > max {
		return fmt.Sprintf("%q...[truncated %d bytes]", b[:max], len(b)-max)
	}
	return fmt.Sprintf("%q", b)
}

// SafePath escapes each segment of a "/"-separated repository path so it
// cannot alter the request path or query. Empty input returns "".
func SafePath(p string) (string, error) {
	if p == "" {
		return "", nil
	}
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if s == "" || s == "." || s == ".." {
			return "", fmt.Errorf("invalid path %q", p)
		}
		segs[i] = url.PathEscape(s)
	}
	return strings.Join(segs, "/"), nil
}
