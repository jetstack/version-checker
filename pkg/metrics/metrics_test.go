package metrics

import (
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	m := New(logrus.NewEntry(logrus.New()))

	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		m.AddImage("namespace", "pod", "container", typ, "url", true, version, version)
	}

	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		mt, _ := m.containerImageVersion.GetMetricWith(m.buildLabels("namespace", "pod", "container", typ, "url", version, version))
		count := testutil.ToFloat64(mt)
		if count != 1 {
			t.Error("Should have added metric")
		}
	}

	for _, typ := range []string{"init", "container"} {
		m.RemoveImage("namespace", "pod", "container", typ)
	}
	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		mt, _ := m.containerImageVersion.GetMetricWith(m.buildLabels("namespace", "pod", "container", typ, "url", version, version))
		count := testutil.ToFloat64(mt)
		if count != 0 {
			t.Error("Should have removed metric")
		}
	}
}

// TestErrorsReporting verifies that the error metric increments correctly
func TestErrorsReporting(t *testing.T) {
	m := New(logrus.NewEntry(logrus.New()))

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
			m.ErrorsReporting(tc.namespace, tc.pod, tc.container, tc.image)

			// Retrieve metric
			metric, err := m.containerImageErrors.GetMetricWith(m.buildPartialLabels(
				tc.namespace,
				tc.pod,
			))
			assert.NoError(t, err, "Failed to get metric with labels")

			// Validate metric count
			fetchErrorCount := testutil.ToFloat64(metric)
			assert.Equal(t, float64(tc.expected), fetchErrorCount, "Expected error count to increment correctly")
		})
	}
}
