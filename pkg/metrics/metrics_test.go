package metrics

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
)

var fakek8s = fake.NewFakeClient()

func TestCache(t *testing.T) {
	m := New(logrus.NewEntry(logrus.New()), prometheus.NewRegistry(), fakek8s)

	// Lets add some Images/Metrics...
	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		m.AddImage("namespace", "pod", "container", typ, "url", true, version, version)
	}

	// Check and ensure that the metrics are available...
	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		mt, _ := m.containerImageVersion.GetMetricWith(
			buildFullLabels("namespace", "pod", "container", typ, "url", version, version),
		)
		count := testutil.ToFloat64(mt)
		assert.Equal(t, count, float64(1), "Expected to get a metric for containerImageVersion")
	}

	// as well as the lastUpdated...
	for _, typ := range []string{"init", "container"} {
		mt, err := m.containerImageChecked.GetMetricWith(buildLastUpdatedLabels("namespace", "pod", "container", typ, "url"))
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
		mt, _ := m.containerImageVersion.GetMetricWith(
			buildFullLabels("namespace", "pod", "container", typ, "url", version, version),
		)
		count := testutil.ToFloat64(mt)
		assert.Equal(t, count, float64(0), "Expected NOT to get a metric for containerImageVersion")
	}
	// And the Last Updated is removed too
	for _, typ := range []string{"init", "container"} {
		mt, err := m.containerImageChecked.GetMetricWith(buildLastUpdatedLabels("namespace", "pod", "container", typ, "url"))
		require.NoError(t, err)
		count := testutil.ToFloat64(mt)
		assert.Equal(t, count, float64(0), "Expected to get a metric for containerImageChecked")
	}
}

// TestErrorsReporting verifies that the error metric increments correctly
func TestErrorsReporting(t *testing.T) {
	m := New(logrus.NewEntry(logrus.New()), prometheus.NewRegistry(), fakek8s)

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
			err := fakek8s.DeleteAllOf(context.Background(), &corev1.Pod{})
			require.NoError(t, err)

			// We need to ensure that the pod Exists!
			err = fakek8s.Create(context.Background(), &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: tc.pod, Namespace: tc.namespace},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: tc.container, Image: tc.image}}},
			})
			require.NoError(t, err)

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

func Test_Metrics_SkipOnDeletedPod(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Step 1: Create fake client with Pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mypod",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build()

	// Step 2: Create Metrics with fake registry
	reg := prometheus.NewRegistry()
	log := logrus.NewEntry(logrus.New())
	metrics := New(log, reg, client)

	// verify Pod exists!
	require.True(t,
		metrics.PodExists(context.Background(), "default", "mypod"),
		"Pod should exist at this point!",
	)

	// Register some metrics....
	metrics.RegisterImageDuration("default", "mypod", "mycontainer", "nginx:latest", time.Now())

	// Step 3: Simulate a Delete occuring, Whilst still Reconciling...
	_ = client.Delete(context.Background(), pod)
	metrics.RemovePod("default", "mypod")

	// Step 4: Validate that all metrics have been removed...
	metricFamilies, err := reg.Gather()
	assert.NoError(t, err)
	for _, mf := range metricFamilies {
		assert.NotContains(t, *mf.Name, "is_latest_version", "Should not have been found: %+v", mf)
		assert.NotContains(t, *mf.Name, "image_lookup_duration", "Should not have been found: %+v", mf)
		assert.NotContains(t, *mf.Name, "image_failures_total", "Should not have been found: %+v", mf)
	}

	// Register Error _after_ sync has completed!
	metrics.ReportError("default", "mypod", "mycontianer", "nginx:latest")

	// Step 5: Attempt to register metrics (should not register anything)
	require.False(t,
		metrics.PodExists(context.Background(), "default", "mypod"),
		"Pod should NOT exist at this point!",
	)

	metrics.RegisterImageDuration("default", "mypod", "mycontainer", "nginx:latest", time.Now())
	metrics.ReportError("default", "mypod", "mycontianer", "nginx:latest")

	// Step 6: Gather metrics and assert none were registered
	metricFamilies, err = reg.Gather()
	assert.NoError(t, err)
	for _, mf := range metricFamilies {
		assert.NotContains(t, *mf.Name, "is_latest_version", "Should not have been found: %+v", mf)
		assert.NotContains(t, *mf.Name, "image_lookup_duration", "Should not have been found: %+v", mf)
		assert.NotContains(t, *mf.Name, "image_failures_total", "Should not have been found: %+v", mf)
	}
}

func TestPodAnnotationsChangeAfterRegistration(t *testing.T) {
	// Step 2: Create Metrics with fake registry
	reg := prometheus.NewRegistry()
	log := logrus.NewEntry(logrus.New())
	client := fake.NewClientBuilder().Build()
	metrics := New(log, reg, client)

	// Register Metrics...
	metrics.AddImage("default", "mypod", "my-init-container", "init", "alpine:latest", false, "1.0", "1.1")
	metrics.AddImage("default", "mypod", "mycontainer", "container", "nginx:1.0", true, "1.0", "1.0")
	metrics.AddImage("default", "mypod", "sidecar", "container", "alpine:1.0", false, "1.0", "1.1")

	_, err := reg.Gather()
	require.NoError(t, err, "Failed to gather metrics")

	assert.Equal(t, 3,
		testutil.CollectAndCount(metrics.containerImageVersion.MetricVec, MetricNamespace+"_is_latest_version"),
	)

	// Pod Annotations are changed, only the `mycontainer` should be checked...

	// Remove Init and sidecar
	metrics.RemoveImage("default", "mypod", "my-init-container", "init")
	metrics.RemoveImage("default", "mypod", "sidecar", "container")

	assert.Equal(t, 1,
		testutil.CollectAndCount(metrics.containerImageVersion.MetricVec, MetricNamespace+"_is_latest_version"),
	)
}
