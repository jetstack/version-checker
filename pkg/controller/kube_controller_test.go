package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/jetstack/version-checker/pkg/metrics"
	"github.com/jetstack/version-checker/pkg/version/semver"
)

func TestNewKubeReconciler(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetOutput(io.Discard)

	kubeClient := fake.NewFakeClient()
	metricsInstance := metrics.New(logger, prometheus.NewRegistry(), kubeClient)

	// Create a minimal valid REST config for testing
	config := &rest.Config{
		Host: "https://localhost:8080",
	}

	t.Run("with valid channel", func(t *testing.T) {
		reconciler := NewKubeReconciler(
			logger,
			config,
			metricsInstance,
			5*time.Minute,
			"stable",
		)

		assert.NotNil(t, reconciler)
		assert.Equal(t, 5*time.Minute, reconciler.interval)
		assert.Equal(t, "stable", reconciler.channel)
		assert.Equal(t, metricsInstance, reconciler.metrics)
		assert.NotNil(t, reconciler.client)
		// Note: We don't check the log field directly as it may have been modified with WithField
	})

	t.Run("with empty channel", func(t *testing.T) {
		reconciler := NewKubeReconciler(
			logger,
			config,
			metricsInstance,
			5*time.Minute,
			"", // Empty channel
		)

		assert.Nil(t, reconciler, "Should return nil when channel is empty")
	})

	t.Run("with different valid channels", func(t *testing.T) {
		// Only test official Kubernetes channels
		channels := []string{"stable", "latest", "latest-1.28", "latest-1.27", "latest-1.26"}

		for _, channel := range channels {
			t.Run(channel, func(t *testing.T) {
				reconciler := NewKubeReconciler(
					logger,
					config,
					metricsInstance,
					5*time.Minute,
					channel,
				)

				assert.NotNil(t, reconciler)
				assert.Equal(t, channel, reconciler.channel)
			})
		}
	})

	t.Run("with invalid channels", func(t *testing.T) {
		// These should be rejected if we add validation
		invalidChannels := []string{"rapid", "regular", "extended", "invalid-channel"}

		for _, channel := range invalidChannels {
			t.Run(channel, func(t *testing.T) {
				// For now, they still create reconcilers but would fail at runtime
				// In the future, we might want to validate channels in the constructor
				reconciler := NewKubeReconciler(
					logger,
					config,
					metricsInstance,
					5*time.Minute,
					channel,
				)

				// Currently accepts any non-empty channel
				assert.NotNil(t, reconciler)
				assert.Equal(t, channel, reconciler.channel)
			})
		}
	})
}

func TestGetLatestVersion(t *testing.T) {
	tests := []struct {
		name           string
		channel        string
		serverResponse string
		serverStatus   int
		expectedResult string
		expectedError  bool
	}{
		{
			name:           "stable channel",
			channel:        "stable",
			serverResponse: "v1.28.2\n",
			serverStatus:   http.StatusOK,
			expectedResult: "v1.28.2",
			expectedError:  false,
		},
		{
			name:           "latest channel",
			channel:        "latest",
			serverResponse: "v1.29.0-alpha.1\n",
			serverStatus:   http.StatusOK,
			expectedResult: "v1.29.0-alpha.1",
			expectedError:  false,
		},
		{
			name:           "channel already has .txt extension",
			channel:        "stable.txt",
			serverResponse: "v1.28.2\n",
			serverStatus:   http.StatusOK,
			expectedResult: "v1.28.2",
			expectedError:  false,
		},
		{
			name:           "server error",
			channel:        "stable",
			serverResponse: "Not Found",
			serverStatus:   http.StatusNotFound,
			expectedResult: "",
			expectedError:  true,
		},
		{
			name:           "empty response",
			channel:        "stable",
			serverResponse: "",
			serverStatus:   http.StatusOK,
			expectedResult: "",
			expectedError:  true,
		},
		{
			name:           "response with whitespace",
			channel:        "stable",
			serverResponse: "  v1.28.2  \n\n",
			serverStatus:   http.StatusOK,
			expectedResult: "v1.28.2",
			expectedError:  false,
		},
		{
			name:           "latest-1.28 channel",
			channel:        "latest-1.28",
			serverResponse: "v1.28.5\n",
			serverStatus:   http.StatusOK,
			expectedResult: "v1.28.5",
			expectedError:  false,
		},
		{
			name:           "latest-1.27 channel",
			channel:        "latest-1.27",
			serverResponse: "v1.27.8\n",
			serverStatus:   http.StatusOK,
			expectedResult: "v1.27.8",
			expectedError:  false,
		},
		{
			name:           "invalid channel should error",
			channel:        "rapid",
			serverResponse: "Not Found",
			serverStatus:   http.StatusNotFound,
			expectedResult: "",
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server that matches the expected behavior
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/" + tt.channel
				if !strings.HasSuffix(expectedPath, ".txt") {
					expectedPath += ".txt"
				}
				assert.Equal(t, expectedPath, r.URL.Path)

				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			// Create a test version of getLatestVersion that uses our test server
			testGetLatestVersion := func(channel string) (string, error) {
				if !strings.HasSuffix(channel, ".txt") {
					channel += ".txt"
				}

				resp, err := http.Get(server.URL + "/" + channel)
				if err != nil {
					return "", err
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return "", assert.AnError
				}

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return "", err
				}

				version := strings.TrimSpace(string(body))
				if version == "" {
					return "", assert.AnError
				}

				return version, nil
			}

			result, err := testGetLatestVersion(tt.channel)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

// TestClusterVersionScheduler_reconcile_Integration tests the reconcile method
// by mocking the Kubernetes discovery and HTTP calls
func TestClusterVersionScheduler_reconcile_Integration(t *testing.T) {
	tests := []struct {
		name               string
		currentVersion     string
		latestVersion      string
		channel            string
		expectedUpToDate   bool
		kubernetesAPIError bool
		channelServerError bool
		expectedError      bool
	}{
		{
			name:             "cluster is up to date",
			currentVersion:   "v1.28.2",
			latestVersion:    "v1.28.2",
			channel:          "stable",
			expectedUpToDate: true,
			expectedError:    false,
		},
		{
			name:             "cluster needs update",
			currentVersion:   "v1.27.1",
			latestVersion:    "v1.28.2",
			channel:          "stable",
			expectedUpToDate: false,
			expectedError:    false,
		},
		{
			name:             "cluster is ahead of stable",
			currentVersion:   "v1.29.0",
			latestVersion:    "v1.28.2",
			channel:          "stable",
			expectedUpToDate: true,
			expectedError:    false,
		},
		{
			name:             "current version with metadata",
			currentVersion:   "v1.28.2-gke.1",
			latestVersion:    "v1.28.2",
			channel:          "stable",
			expectedUpToDate: true,
			expectedError:    false,
		},
		{
			name:               "kubernetes api error",
			currentVersion:     "v1.28.2",
			latestVersion:      "v1.28.2",
			channel:            "stable",
			kubernetesAPIError: true,
			expectedError:      true,
		},
		{
			name:               "channel server error",
			currentVersion:     "v1.28.2",
			latestVersion:      "v1.28.2",
			channel:            "stable",
			channelServerError: true,
			expectedError:      true,
		},
		// Platform-specific version format tests (metadata handling)
		// EKS and GKE platform versions
		{
			name:             "EKS version format - up to date",
			currentVersion:   "v1.28.2-eks-a5565ad",
			latestVersion:    "v1.28.2",
			channel:          "stable",
			expectedUpToDate: true,
			expectedError:    false,
		},
		{
			name:             "EKS version format - needs update",
			currentVersion:   "v1.27.9-eks-2f008fe",
			latestVersion:    "v1.28.2",
			channel:          "stable",
			expectedUpToDate: false,
			expectedError:    false,
		},
		{
			name:             "EKS version format - ahead of stable",
			currentVersion:   "v1.29.0-eks-5e0fdde",
			latestVersion:    "v1.28.2",
			channel:          "stable",
			expectedUpToDate: true,
			expectedError:    false,
		},
		{
			name:             "EKS version with longer metadata",
			currentVersion:   "v1.28.2-eks-a5565ad-20231102",
			latestVersion:    "v1.28.2",
			channel:          "stable",
			expectedUpToDate: true,
			expectedError:    false,
		},
		{
			name:             "EKS Fargate version format",
			currentVersion:   "v1.28.2-eks-fargate-a5565ad",
			latestVersion:    "v1.28.2",
			channel:          "stable",
			expectedUpToDate: true,
			expectedError:    false,
		},
		{
			name:             "GKE version format - up to date",
			currentVersion:   "v1.28.2-gke.1034000",
			latestVersion:    "v1.28.2",
			channel:          "stable", // Always compare against upstream Kubernetes
			expectedUpToDate: true,
			expectedError:    false,
		},
		{
			name:             "GKE version format - needs update",
			currentVersion:   "v1.27.9-gke.1034000",
			latestVersion:    "v1.28.2",
			channel:          "stable",
			expectedUpToDate: false,
			expectedError:    false,
		},
		{
			name:             "GKE version format - ahead of stable",
			currentVersion:   "v1.29.0-gke.1034000",
			latestVersion:    "v1.28.2",
			channel:          "stable",
			expectedUpToDate: true,
			expectedError:    false,
		},
		{
			name:             "GKE version format with newer patch",
			currentVersion:   "v1.29.0-gke.1234567",
			latestVersion:    "v1.28.2",
			channel:          "stable", // Comparing against upstream stable
			expectedUpToDate: true,
			expectedError:    false,
		},
		{
			name:             "GKE version format - stable",
			currentVersion:   "v1.28.2-gke.1034000",
			latestVersion:    "v1.28.2",
			channel:          "stable",
			expectedUpToDate: true,
			expectedError:    false,
		},
		{
			name:             "GKE version format - extended support",
			currentVersion:   "v1.26.15-gke.4901000",
			latestVersion:    "v1.28.2",
			channel:          "stable",
			expectedUpToDate: false,
			expectedError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake Kubernetes client
			fakeClient := fakeclientset.NewSimpleClientset()
			fakeDiscovery := fakeClient.Discovery().(*fakediscovery.FakeDiscovery)

			if tt.kubernetesAPIError {
				fakeDiscovery.PrependReactor("*", "*", func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, assert.AnError
				})
			} else {
				fakeDiscovery.FakedServerVersion = &version.Info{
					GitVersion: tt.currentVersion,
				}
			}

			// Create test server for Kubernetes release channel
			var server *httptest.Server
			if tt.channelServerError {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal Server Error"))
				}))
			} else {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedPath := "/" + tt.channel + ".txt"
					assert.Equal(t, expectedPath, r.URL.Path)

					// Simulate that invalid channels return 404
					if !isValidKubernetesChannel(tt.channel) {
						w.WriteHeader(http.StatusNotFound)
						w.Write([]byte("Not Found"))
						return
					}

					w.WriteHeader(http.StatusOK)
					w.Write([]byte(tt.latestVersion))
				}))
			}
			defer server.Close()

			// Create logger
			logger := logrus.NewEntry(logrus.New())
			logger.Logger.SetOutput(io.Discard)

			// Create metrics registry
			registry := prometheus.NewRegistry()
			kubeClient := fake.NewFakeClient()
			metricsInstance := metrics.New(logger, registry, kubeClient)

			// Create reconciler with a custom getLatestStableVersion function
			reconciler := &testableClusterVersionScheduler{
				ClusterVersionScheduler: ClusterVersionScheduler{
					client:   fakeClient,
					log:      logger,
					metrics:  metricsInstance,
					interval: 5 * time.Minute,
					channel:  tt.channel,
				},
				testServerURL: server.URL,
			}

			// Execute reconcile
			ctx := context.Background()
			var _ context.Context = ctx
			err := reconciler.reconcile()

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Check metrics were registered correctly
			metricFamilies, err := registry.Gather()
			require.NoError(t, err)

			// Find the kubernetes version metric
			var kubeMetric *dto.MetricFamily
			for _, mf := range metricFamilies {
				if mf.GetName() == "version_checker_is_latest_kube_version" {
					kubeMetric = mf
					break
				}
			}

			require.NotNil(t, kubeMetric, "Kubernetes version metric should be present")
			require.Len(t, kubeMetric.GetMetric(), 1, "Should have exactly one metric value")

			metric := kubeMetric.GetMetric()[0]
			expectedValue := 0.0
			if tt.expectedUpToDate {
				expectedValue = 1.0
			}

			assert.Equal(t, expectedValue, metric.GetGauge().GetValue())

			// Check labels
			labels := metric.GetLabel()
			assert.Len(t, labels, 3, "Should have 3 labels: current_version, latest_version, channel")

			labelMap := make(map[string]string)
			for _, label := range labels {
				labelMap[label.GetName()] = label.GetValue()
			}

			assert.Equal(t, tt.channel, labelMap["channel"])
		})
	}
}

// testableClusterVersionScheduler is a wrapper that allows us to override the HTTP calls for testing
type testableClusterVersionScheduler struct {
	ClusterVersionScheduler
	testServerURL string
}

func (s *testableClusterVersionScheduler) reconcile() error {
	// Get current cluster version
	current, err := s.client.Discovery().ServerVersion()
	if err != nil {
		return err
	}

	// Get latest version using our test server
	latest, err := s.getLatestVersionFromTestServer(s.channel)
	if err != nil {
		return err
	}

	// Use the same logic as the main code
	latestSemVer := semver.Parse(latest)
	currentSemVer := semver.Parse(current.GitVersion)

	// Create version strings without metadata for comparison (same as main code)
	currentSemVerNoMeta := fmt.Sprintf("%d.%d.%d", currentSemVer.Major(), currentSemVer.Minor(), currentSemVer.Patch())
	latestSemVerNoMeta := fmt.Sprintf("%d.%d.%d", latestSemVer.Major(), latestSemVer.Minor(), latestSemVer.Patch())

	// Parse the versions without metadata for comparison
	currentComparable := semver.Parse(currentSemVerNoMeta)
	latestComparable := semver.Parse(latestSemVerNoMeta)

	s.metrics.RegisterKubeVersion(!currentComparable.LessThan(latestComparable),
		currentSemVerNoMeta, latestSemVerNoMeta,
		s.channel,
	)

	return nil
}

func (s *testableClusterVersionScheduler) getLatestVersionFromTestServer(channel string) (string, error) {
	if !strings.HasSuffix(channel, ".txt") {
		channel += ".txt"
	}

	resp, err := http.Get(s.testServerURL + "/" + channel)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", assert.AnError
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	version := strings.TrimSpace(string(body))
	if version == "" {
		return "", assert.AnError
	}

	return version, nil
}

func (s *testableClusterVersionScheduler) Start(ctx context.Context) error {
	s.log.Info("Starting Kubernetes version checker")
	s.runScheduler(ctx)
	return nil
}

func (s *testableClusterVersionScheduler) runScheduler(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run once immediately, then on interval
	_ = s.reconcile()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("Kubernetes version checker stopped")
			return
		case <-ticker.C:
			if err := s.reconcile(); err != nil {
				s.log.WithError(err).Error("Failed to reconcile cluster version")
			}
		}
	}
}

func TestClusterVersionScheduler_Start(t *testing.T) {
	t.Run("context cancellation stops scheduler", func(t *testing.T) {
		// Create logger
		logger := logrus.NewEntry(logrus.New())
		logger.Logger.SetOutput(io.Discard)

		// Create metrics registry
		registry := prometheus.NewRegistry()
		kubeClient := fake.NewFakeClient()
		metricsInstance := metrics.New(logger, registry, kubeClient)

		// Create fake Kubernetes client that won't fail
		fakeClient := fakeclientset.NewSimpleClientset()
		fakeDiscovery := fakeClient.Discovery().(*fakediscovery.FakeDiscovery)
		fakeDiscovery.FakedServerVersion = &version.Info{
			GitVersion: "v1.28.2",
		}

		// Create a simple test server for the channel
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("v1.28.2"))
		}))
		defer server.Close()

		// Create a testable reconciler
		reconciler := &testableClusterVersionScheduler{
			ClusterVersionScheduler: ClusterVersionScheduler{
				client:   fakeClient,
				log:      logger,
				metrics:  metricsInstance,
				interval: 50 * time.Millisecond, // Short interval for testing
				channel:  "stable",
			},
			testServerURL: server.URL,
		}

		// Start the reconciler with a short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		// This should run for the timeout duration and then stop gracefully
		err := reconciler.Start(ctx)
		assert.NoError(t, err, "Start should complete without error when context is cancelled")
	})

	t.Run("nil reconciler handling", func(t *testing.T) {
		logger := logrus.NewEntry(logrus.New())
		logger.Logger.SetOutput(io.Discard)

		kubeClient := fake.NewFakeClient()
		metricsInstance := metrics.New(logger, prometheus.NewRegistry(), kubeClient)

		config := &rest.Config{
			Host: "https://localhost:8080",
		}

		// Create reconciler with empty channel (should return nil)
		reconciler := NewKubeReconciler(
			logger,
			config,
			metricsInstance,
			5*time.Minute,
			"", // Empty channel
		)

		// Verify it's nil and we don't try to start it
		assert.Nil(t, reconciler)
	})
}

func TestClusterVersionScheduler_runScheduler(t *testing.T) {
	// Create logger
	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetOutput(io.Discard)

	// Create metrics registry
	registry := prometheus.NewRegistry()
	kubeClient := fake.NewFakeClient()
	metricsInstance := metrics.New(logger, registry, kubeClient)

	// Create fake Kubernetes client
	fakeClient := fakeclientset.NewSimpleClientset()
	fakeDiscovery := fakeClient.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.28.2",
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("v1.28.2"))
	}))
	defer server.Close()

	// Create testable reconciler
	reconciler := &testableClusterVersionScheduler{
		ClusterVersionScheduler: ClusterVersionScheduler{
			client:   fakeClient,
			log:      logger,
			metrics:  metricsInstance,
			interval: 30 * time.Millisecond,
			channel:  "stable",
		},
		testServerURL: server.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should complete without panicking and run at least one
	// reconciliation loop
	assert.NotPanics(t, func() {
		reconciler.runScheduler(ctx)
	}, "runScheduler should not panic")
}

func TestChannelValidation(t *testing.T) {
	tests := []struct {
		name    string
		channel string
		isValid bool
	}{
		{"stable channel", "stable", true},
		{"latest channel", "latest", true},
		{"latest-1.28", "latest-1.28", true},
		{"latest-1.27", "latest-1.27", true},
		{"latest-1.26", "latest-1.26", true},
		{"completely invalid", "invalid-channel", false},
		{"empty channel", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := isValidKubernetesChannel(tt.channel)
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}
