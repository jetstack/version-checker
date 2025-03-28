package controller

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client"
	"github.com/jetstack/version-checker/pkg/controller/checker"
	"github.com/jetstack/version-checker/pkg/controller/options"
	"github.com/jetstack/version-checker/pkg/controller/search"
	"github.com/jetstack/version-checker/pkg/metrics"
	"github.com/jetstack/version-checker/pkg/version"
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
