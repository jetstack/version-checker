package metrics

import (
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	m := New(logrus.NewEntry(logrus.New()), prometheus.NewRegistry())

	// Lets add some Images/Metrics...
	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		m.AddImage("namespace", "pod", "container", typ, "url", true, version, version)
	}

	// Check and ensure that the metrics are available...
	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		mt, err := m.containerImageVersion.GetMetricWith(m.buildFullLabels("namespace", "pod", "container", typ, "url", version, version))
		require.NoError(t, err)
		count := testutil.ToFloat64(mt)
		assert.Equal(t, count, float64(1), "Expected to get a metric for containerImageVersion")
	}

	// as well as the lastUpdated...
	for _, typ := range []string{"init", "container"} {
		mt, err := m.containerImageChecked.GetMetricWith(m.buildLastUpdatedLabels("namespace", "pod", "container", typ, "url"))
		require.NoError(t, err)
		count := testutil.ToFloat64(mt)
		assert.GreaterOrEqual(t, count, float64(time.Now().Unix()))
	}

	// Remove said metrics...
	for _, typ := range []string{"init", "container"} {
		m.RemoveImage("namespace", "pod", "container", typ)
	}
	// Ensure metrics and values return 0
	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		mt, err := m.containerImageVersion.GetMetricWith(m.buildFullLabels("namespace", "pod", "container", typ, "url", version, version))
		require.NoError(t, err)
		count := testutil.ToFloat64(mt)
		assert.Equal(t, count, float64(0), "Expected to get a metric for containerImageVersion")
	}
	// And the Last Updated is removed too
	for _, typ := range []string{"init", "container"} {
		mt, err := m.containerImageChecked.GetMetricWith(m.buildLastUpdatedLabels("namespace", "pod", "container", typ, "url"))
		require.NoError(t, err)
		count := testutil.ToFloat64(mt)
		assert.Equal(t, count, float64(0), "Expected to get a metric for containerImageChecked")
	}
}

// TestErrorsReporting verifies that the error metric increments correctly
func TestErrorsReporting(t *testing.T) {
	m := New(logrus.NewEntry(logrus.New()), prometheus.NewRegistry())

	// Reset the metrics before testing
	m.containerImageErrors.Reset()

	testCases := []struct {
		namespace string
		pod       string
		container string
		image     string
		expected  int
	}{
		{"namespace", "pod", "container", "url", 1},
		{"namespace", "pod", "container", "url", 2},
		{"namespace2", "pod2", "container2", "url2", 1},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Case %d", i+1), func(t *testing.T) {
			// Report an error
			m.ReportError(tc.namespace, tc.pod, tc.container, tc.image)

			// Retrieve metric
			metric, err := m.containerImageErrors.GetMetricWith(prometheus.Labels{
				"namespace": tc.namespace,
				"pod":       tc.pod,
				"container": tc.container,
				"image":     tc.image,
			})
			require.NoError(t, err, "Failed to get metric with labels")

			// Validate metric count
			fetchErrorCount := testutil.ToFloat64(metric)
			assert.Equal(t, float64(tc.expected), fetchErrorCount, "Expected error count to increment correctly")
		})
	}
}
