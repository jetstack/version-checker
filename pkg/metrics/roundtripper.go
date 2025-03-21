package metrics

import (
	"crypto/tls"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	clientInFlightGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      "client_in_flight_requests",
		Help:      "A gauge of in-flight requests for the wrapped client.",
		Namespace: "http",
	})

	clientCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "client_requests_total",
			Help:      "A counter for requests from the wrapped client.",
			Namespace: "http",
		},
		[]string{"code", "method", "domain"}, // Ensure domain is explicitly part of the label definition
	)

	histVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "client_request_duration_seconds",
			Help:      "A histogram of request durations.",
			Namespace: "http",
		},
		[]string{"method", "domain"}, // Explicit labels
	)

	tlsLatencyVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "tls_duration_seconds",
			Help:      "Trace TLS latency histogram.",
			Namespace: "http",
		},
		[]string{"event", "domain"},
	)

	dnsLatencyVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "dns_duration_seconds",
			Help:      "Trace DNS latency histogram.",
			Namespace: "http",
		},
		[]string{"event", "domain"},
	)
)

// extractDomain extracts the domain (TLD) from the request URL.
func extractDomain(req *http.Request) string {
	if req.URL == nil {
		return "unknown"
	}
	parsedURL, err := url.Parse(req.URL.String())
	if err != nil {
		return "unknown"
	}
	return parsedURL.Hostname()
}

// tracingRoundTripper wraps RoundTripper to track request metrics.
type tracingRoundTripper struct {
	base http.RoundTripper
}

func (t *tracingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	domain := extractDomain(req)

	// Track request duration
	startTime := time.Now()

	// Track DNS and TLS latencies
	var dnsStart, dnsEnd, tlsStart, tlsEnd time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			dnsEnd = time.Now()
			dnsLatencyVec.WithLabelValues("dns_done", domain).Set(dnsEnd.Sub(dnsStart).Seconds())
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			tlsEnd = time.Now()
			tlsLatencyVec.WithLabelValues("tls_done", domain).Set(tlsEnd.Sub(tlsStart).Seconds())
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	// Perform the request
	resp, err := t.base.RoundTrip(req)

	// Manually record request duration
	histVec.WithLabelValues(req.Method, domain).Set(time.Since(startTime).Seconds())

	if err != nil {
		// In case of failure, still increment counter
		clientCounter.WithLabelValues("error", req.Method, domain).Inc()
		return nil, err
	}

	// Increment counter with domain label
	clientCounter.WithLabelValues(http.StatusText(resp.StatusCode), req.Method, domain).Inc()

	return resp, nil
}

// RoundTripper provides Prometheus instrumentation for an HTTP client, including domain labels.
func RoundTripper(baseTransport http.RoundTripper) http.RoundTripper {
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}

	return promhttp.InstrumentRoundTripperInFlight(clientInFlightGauge,
		&tracingRoundTripper{base: baseTransport}, // Removed InstrumentRoundTripperCounter
	)
}
