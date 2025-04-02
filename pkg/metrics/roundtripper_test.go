package metrics

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/transport"
	"github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var log = logrus.NewEntry(logrus.StandardLogger())

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name     string
		req      *http.Request
		expected string
	}{
		{
			name: "valid URL with domain",
			req: &http.Request{
				URL: &url.URL{
					Scheme: "https",
					Host:   "example.com",
				},
			},
			expected: "example.com",
		},
		{
			name: "valid URL with subdomain",
			req: &http.Request{
				URL: &url.URL{
					Scheme: "https",
					Host:   "sub.example.com",
				},
			},
			expected: "sub.example.com",
		},
		{
			name: "URL with port",
			req: &http.Request{
				URL: &url.URL{
					Scheme: "https",
					Host:   "example.com:8080",
				},
			},
			expected: "example.com",
		},
		{
			name:     "nil URL",
			req:      &http.Request{},
			expected: "unknown",
		},
		{
			name: "URL with port",
			req: &http.Request{
				URL: &url.URL{
					Scheme: "https",
					Host:   "with-port:8443",
				},
			},
			expected: "with-port",
		},
		{
			name: "invalid URL",
			req: &http.Request{
				URL: &url.URL{
					Scheme: "https",
					Host:   "invalid-url",
				},
			},
			expected: "invalid-url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain := extractDomain(tt.req)
			assert.Equal(t, tt.expected, domain)
		})
	}
}

func TestRoundTripper(t *testing.T) {
	// t.Skipf("Still need to fix these")
	t.Parallel()

	tests := []struct {
		name                 string
		handler              http.HandlerFunc
		expectedStatus       int
		expectedError        bool
		expectedMetricString string
	}{
		{
			name: "successful request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectedMetricString: `
				# HELP http_client_in_flight_requests A gauge of in-flight requests for the wrapped client.
				# TYPE http_client_in_flight_requests gauge
				http_client_in_flight_requests 0
				# HELP http_client_request_duration_seconds A histogram of request durations.
				# TYPE http_client_request_duration_seconds gauge
				http_client_request_duration_seconds{domain="127.0.0.1",method="GET"} 0
				# HELP http_client_requests_total A counter for requests from the wrapped client.
				# TYPE http_client_requests_total counter
				http_client_requests_total{code="OK",domain="127.0.0.1",method="GET"} 1
		`,
		},
		{
			name: "failed request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  false,
			expectedMetricString: `
				# HELP http_client_in_flight_requests A gauge of in-flight requests for the wrapped client.
				# TYPE http_client_in_flight_requests gauge
				http_client_in_flight_requests 0
				# HELP http_client_request_duration_seconds A histogram of request durations.
				# TYPE http_client_request_duration_seconds gauge
				http_client_request_duration_seconds{domain="127.0.0.1",method="GET"} 0
				# HELP http_client_requests_total A counter for requests from the wrapped client.
				# TYPE http_client_requests_total counter
				http_client_requests_total{code="Internal Server Error",domain="127.0.0.1",method="GET"} 1
		`,
		},
		{
			name: "request with DNS and TLS latency",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(10 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectedMetricString: `
				# HELP http_client_in_flight_requests A gauge of in-flight requests for the wrapped client.
				# TYPE http_client_in_flight_requests gauge
				http_client_in_flight_requests 0
				# HELP http_client_request_duration_seconds A histogram of request durations.
				# TYPE http_client_request_duration_seconds gauge
				http_client_request_duration_seconds{domain="127.0.0.1",method="GET"} 0
				# HELP http_client_requests_total A counter for requests from the wrapped client.
				# TYPE http_client_requests_total counter
				http_client_requests_total{code="OK",domain="127.0.0.1",method="GET"} 1
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricsServer := New(log, prometheus.NewRegistry(), fakek8s)
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := &http.Client{
				Transport: transport.Chain(http.DefaultTransport, metricsServer.RoundTripper),
			}

			req, err := http.NewRequest("GET", server.URL, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			defer func() { _ = resp.Body.Close() }()

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			}

			// Validate metrics
			assert.NoError(t,
				testutil.GatherAndCompare(
					metricsServer.registry, strings.NewReader(tt.expectedMetricString),
					"http_client_in_flight_requests",
					"http_client_requests_total",
					// "http_client_request_duration_seconds",
					"http_tls_duration_seconds",
					"http_dns_duration_seconds",
				))
		})
	}
}
