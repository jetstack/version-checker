package util

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"
)

var (
	cgnat = netip.MustParsePrefix("100.64.0.0/10")
	nat64 = netip.MustParsePrefix("64:ff9b::/96")
	// Azure's wire server is link-local in spirit but not in range.
	azureWire = netip.MustParseAddr("168.63.129.16")
)

func isInternal(ip netip.Addr) bool {
	ip = ip.Unmap()
	if nat64.Contains(ip) {
		// IPv6-only clusters reach public IPv4 hosts through NAT64: judge the embedded address.
		b := ip.As16()
		ip = netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]})
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() ||
		cgnat.Contains(ip) || ip == azureWire
}

// CheckPublicHost returns an error if host (optionally host:port) is, or
// resolves to, an internal address. It is a pre-flight check; use PublicOnly
// and PublicDialContext to enforce the same policy at connect time.
func CheckPublicHost(ctx context.Context, host string) error {
	if err := ValidHost(host); err != nil {
		return err
	}
	h := host
	if hh, _, err := net.SplitHostPort(host); err == nil {
		h = hh
	}
	h = strings.Trim(h, "[]")
	if h == "" {
		return errors.New("empty host")
	}
	if ip, err := netip.ParseAddr(h); err == nil {
		if isInternal(ip) {
			return fmt.Errorf("host %q is an internal address", host)
		}
		return nil
	}
	lh := strings.ToLower(h)
	if lh == "localhost" || strings.HasSuffix(lh, ".localhost") || strings.HasSuffix(lh, ".local") ||
		strings.HasSuffix(lh, ".internal") || strings.HasSuffix(lh, ".svc") || strings.Contains(lh, ".svc.") {
		return fmt.Errorf("host %q is an internal name", host)
	}
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", h)
	if err != nil {
		return fmt.Errorf("resolving host %q: %w", host, err)
	}
	for _, a := range addrs {
		if isInternal(a) {
			return fmt.Errorf("host %q resolves to internal address %s", host, a)
		}
	}
	return nil
}

// controlPublicOnly is a net.Dialer.Control hook: it sees the address after
// DNS resolution, so it also covers redirects and DNS rebinding.
func controlPublicOnly(_, address string, _ syscall.RawConn) error {
	h, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip, err := netip.ParseAddr(h)
	if err != nil {
		return err
	}
	if isInternal(ip) {
		return fmt.Errorf("refusing to connect to internal address %s", ip)
	}
	return nil
}

type denyKey struct{}

// DialContext dials like a default net.Dialer but, for requests marked by
// PublicOnly, refuses internal addresses after DNS resolution. That covers
// redirects and DNS rebinding, which the pre-flight CheckPublicHost cannot.
// Behind an egress proxy the proxy does the dialing, so only PublicOnly's host
// check applies.
func DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	if ctx.Value(denyKey{}) != nil {
		probe := &http.Request{URL: &url.URL{Scheme: "https", Host: "registry.invalid"}}
		if p, _ := http.ProxyFromEnvironment(probe); p == nil {
			d.Control = controlPublicOnly
		}
	}
	return d.DialContext(ctx, network, addr)
}

type publicOnly struct{ http.RoundTripper }

// PublicOnly wraps rt so every request, redirects included, passes
// CheckPublicHost and is dialed through DialContext's address check (rt's
// transport must use DialContext), and response bodies are capped at
// MaxResponseBytes.
func PublicOnly(rt http.RoundTripper) http.RoundTripper { return publicOnly{rt} }

func (p publicOnly) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := CheckPublicHost(req.Context(), req.URL.Host); err != nil {
		return nil, err
	}
	resp, err := p.RoundTripper.RoundTrip(req.WithContext(context.WithValue(req.Context(), denyKey{}, true)))
	if err == nil {
		resp.Body = http.MaxBytesReader(nil, resp.Body, MaxResponseBytes)
	}
	return resp, err
}
