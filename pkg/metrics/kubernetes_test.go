package metrics

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var fakeK8sClient = fake.NewFakeClient()

func TestRegisterKubeVersion(t *testing.T) {
	tests := []struct {
		name           string
		isLatest       bool
		currentVersion string
		latestVersion  string
		channel        string
		expectedValue  float64
	}{
		{
			name:           "cluster is up to date",
			isLatest:       true,
			currentVersion: "1.28.2",
			latestVersion:  "1.28.2",
			channel:        "stable",
			expectedValue:  1.0,
		},
		{
			name:           "cluster needs update",
			isLatest:       false,
			currentVersion: "1.27.1",
			latestVersion:  "1.28.2",
			channel:        "stable",
			expectedValue:  0.0,
		},
		{
			name:           "cluster is ahead of stable",
			isLatest:       true,
			currentVersion: "1.29.0",
			latestVersion:  "1.28.2",
			channel:        "stable",
			expectedValue:  1.0,
		},
		{
			name:           "latest channel with pre-release",
			isLatest:       false,
			currentVersion: "1.28.1",
			latestVersion:  "1.29.0-alpha.1",
			channel:        "latest",
			expectedValue:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new metrics instance for each test to avoid interference
			registry := prometheus.NewRegistry()
			m := New(logrus.NewEntry(logrus.New()), registry, fakeK8sClient)

			// Register the Kubernetes version
			m.RegisterKubeVersion(tt.isLatest, tt.currentVersion, tt.latestVersion, tt.channel)

			// Gather metrics
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
			assert.Equal(t, tt.expectedValue, metric.GetGauge().GetValue())

			// Check labels
			labels := metric.GetLabel()
			assert.Len(t, labels, 3, "Should have 3 labels: current_version, latest_version, channel")

			labelMap := make(map[string]string)
			for _, label := range labels {
				labelMap[label.GetName()] = label.GetValue()
			}

			assert.Equal(t, tt.currentVersion, labelMap["current_version"])
			assert.Equal(t, tt.latestVersion, labelMap["latest_version"])
			assert.Equal(t, tt.channel, labelMap["channel"])
		})
	}
}

func TestRegisterKubeVersion_MultipleChannels(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := New(logrus.NewEntry(logrus.New()), registry, fakeK8sClient)

	// Register metrics for different channels
	m.RegisterKubeVersion(true, "1.28.2", "1.28.2", "stable")
	m.RegisterKubeVersion(false, "1.28.2", "1.29.0-alpha.1", "latest")

	// Gather metrics
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
	require.Len(t, kubeMetric.GetMetric(), 2, "Should have exactly two metric values for different channels")

	// Check that both metrics are present
	channels := make(map[string]float64)
	for _, metric := range kubeMetric.GetMetric() {
		labelMap := make(map[string]string)
		for _, label := range metric.GetLabel() {
			labelMap[label.GetName()] = label.GetValue()
		}
		channels[labelMap["channel"]] = metric.GetGauge().GetValue()
	}

	assert.Equal(t, 1.0, channels["stable"], "Stable channel should be up to date")
	assert.Equal(t, 0.0, channels["latest"], "Latest channel should need update")
}
