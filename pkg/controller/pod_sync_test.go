package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client"
	"github.com/jetstack/version-checker/pkg/controller/checker"
	fakesearch "github.com/jetstack/version-checker/pkg/controller/internal/fake/search"
	"github.com/jetstack/version-checker/pkg/controller/options"
	"github.com/jetstack/version-checker/pkg/controller/search"
	"github.com/jetstack/version-checker/pkg/metrics"
	"github.com/jetstack/version-checker/pkg/version"
	versionerrors "github.com/jetstack/version-checker/pkg/version/errors"
)

// Test for the sync method.
func TestController_Sync(t *testing.T) {
	t.Parallel()

	log := logrus.NewEntry(logrus.New())
	metrics := metrics.New(
		logrus.NewEntry(logrus.StandardLogger()),
		prometheus.NewRegistry(),
		fake.NewFakeClient(),
	)
	imageClient := &client.Client{}
	searcher := search.New(log, 5*time.Minute, version.New(log, imageClient, 5*time.Minute))
	checker := checker.New(searcher)

	controller := &PodReconciler{
		Log:            log,
		VersionChecker: checker,
		Metrics:        metrics,
		defaultTestAll: true,
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{Name: "init-container"},
			},
			Containers: []corev1.Container{
				{Name: "main-container"},
			},
		},
	}

	err := controller.sync(context.Background(), pod)
	assert.NoError(t, err)
}

// Test for the syncContainer method.
func TestController_SyncContainer(t *testing.T) {
	t.Parallel()
	log := logrus.NewEntry(logrus.New())
	metrics := metrics.New(log, prometheus.NewRegistry(), fake.NewFakeClient())
	imageClient := &client.Client{}
	searcher := search.New(log, 5*time.Minute, version.New(log, imageClient, 5*time.Minute))
	checker := checker.New(searcher)

	controller := &PodReconciler{
		Log:            log,
		VersionChecker: checker,
		Metrics:        metrics,
		defaultTestAll: true,
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}
	container := &corev1.Container{Name: "main-container"}

	builder := options.New(map[string]string{
		"version-checker.jetstack.io/enabled": "true",
	})

	err := controller.syncContainer(context.Background(), log, builder, pod, container, "container")
	assert.NoError(t, err)
}

// Test for the checkContainer method.
func TestController_CheckContainer(t *testing.T) {
	t.Parallel()
	log := logrus.NewEntry(logrus.New())
	metrics := metrics.New(log, prometheus.NewRegistry(), fake.NewFakeClient())
	imageClient := &client.Client{}
	searcher := search.New(log, 5*time.Minute, version.New(log, imageClient, 5*time.Minute))
	checker := checker.New(searcher)

	controller := &PodReconciler{
		Log:            log,
		VersionChecker: checker,
		Metrics:        metrics,
		defaultTestAll: true,
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}
	container := &corev1.Container{Name: "main-container"}
	opts := &api.Options{}

	err := controller.checkContainer(context.Background(), log, pod, container, "container", opts)
	assert.NoError(t, err)
}

// Example of testing syncContainer when version is not found.
func TestController_SyncContainer_NoVersionFound(t *testing.T) {
	t.Parallel()

	log := logrus.NewEntry(logrus.New())
	metrics := metrics.New(log, prometheus.NewRegistry(), fake.NewFakeClient())
	imageClient := &client.Client{}
	searcher := search.New(log, 5*time.Minute, version.New(log, imageClient, 5*time.Minute))
	checker := checker.New(searcher)

	controller := &PodReconciler{
		Log:            log,
		VersionChecker: checker,
		Metrics:        metrics,

		defaultTestAll: true,
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}
	container := &corev1.Container{Name: "main-container"}
	builder := options.New(map[string]string{
		"version-checker.jetstack.io/enabled": "true",
	})

	err := controller.syncContainer(context.Background(), log, builder, pod, container, "container")
	assert.NoError(t, err) // We expect no error because IsNoVersionFound is handled gracefully
}

func TestController_SyncContainer_NoVersionFoundReportsFailureMetric(t *testing.T) {
	t.Parallel()

	log := logrus.NewEntry(logrus.New())
	reg := prometheus.NewRegistry()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "main-container", Image: "docker.io/example/missing:v1.2.3"},
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "main-container", ImageID: "docker.io/example/missing@sha256:deadbeef"},
			},
		},
	}
	kubeClient := fake.NewClientBuilder().WithObjects(pod).Build()
	metrics := metrics.New(log, reg, kubeClient)
	checker := checker.New(
		fakesearch.New().With(nil, versionerrors.NewVersionErrorNotFound("%s", fmt.Sprintf("no tags found for given image URL: %q", "docker.io/example/missing"))),
	)

	controller := &PodReconciler{
		Log:            log,
		VersionChecker: checker,
		Metrics:        metrics,
		defaultTestAll: true,
	}

	builder := options.New(map[string]string{
		"version-checker.jetstack.io/enabled": "true",
	})

	err := controller.syncContainer(
		context.Background(),
		log,
		builder,
		pod,
		&pod.Spec.Containers[0],
		"container",
	)
	require.NoError(t, err)

	metricFamilies, err := reg.Gather()
	require.NoError(t, err)

	metric := findMetricWithLabels(t, metricFamilies, "version_checker_image_failures_total", map[string]string{
		"namespace": "default",
		"pod":       "test-pod",
		"container": "main-container",
		"image":     "docker.io/example/missing:v1.2.3",
	})
	require.NotNil(t, metric.Counter)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func findMetricWithLabels(t *testing.T, metricFamilies []*dto.MetricFamily, name string, expectedLabels map[string]string) *dto.Metric {
	t.Helper()

	for _, mf := range metricFamilies {
		if mf.GetName() != name {
			continue
		}

		for _, metric := range mf.GetMetric() {
			labels := make(map[string]string, len(metric.GetLabel()))
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if matchesExpectedLabels(labels, expectedLabels) {
				return metric
			}
		}
	}

	require.FailNow(t, fmt.Sprintf("metric %q with labels %+v not found", name, expectedLabels))
	return nil
}

func matchesExpectedLabels(labels, expectedLabels map[string]string) bool {
	for key, value := range expectedLabels {
		if labels[key] != value {
			return false
		}
	}

	return true
}
