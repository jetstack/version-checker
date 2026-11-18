package util

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckPublicHost(t *testing.T) {
	for _, h := range []string{
		"127.0.0.1", "10.0.0.1:5000", "169.254.169.254", "[::1]", "localhost", "x.svc.cluster.local",
		"100.64.0.1", "168.63.129.16", "[64:ff9b::a00:1]", "[::ffff:127.0.0.1]",
		// URL metacharacters must not smuggle an internal host past the check.
		"x@169.254.169.254", "127.0.0.1?x", "127.0.0.1#x", "kubernetes.default.svc?",
	} {
		if CheckPublicHost(context.Background(), h) == nil {
			t.Errorf("%q should be rejected", h)
		}
	}
	// A public IPv4 reached through NAT64 is allowed.
	for _, h := range []string{"8.8.8.8", "[64:ff9b::808:808]"} {
		if err := CheckPublicHost(context.Background(), h); err != nil {
			t.Errorf("%q: %v", h, err)
		}
	}
}

// The httptest server listens on loopback, so a guarded client must refuse it
// both before dialing (PublicOnly) and at connect time (DialContext).
func TestPublicOnlyRefusesLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()

	tr := &http.Transport{DialContext: DialContext}
	if _, err := (&http.Client{Transport: PublicOnly(tr)}).Get(srv.URL); err == nil {
		t.Error("PublicOnly should refuse a loopback host")
	}

	addr := srv.Listener.Addr().String()
	if c, err := DialContext(context.Background(), "tcp", addr); err != nil {
		t.Errorf("unmarked dial should work: %v", err)
	} else {
		_ = c.Close()
	}
	if c, err := DialContext(context.WithValue(context.Background(), denyKey{}, true), "tcp", addr); err == nil {
		_ = c.Close()
		t.Error("marked dial should refuse a loopback address")
	}
}
