package controller

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"github.com/prometheus/client_golang/prometheus"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/jetstack/version-checker/pkg/client"
	"github.com/jetstack/version-checker/pkg/metrics"
)

var testLogger = logrus.NewEntry(logrus.New())

func init() {
	testLogger.Logger.SetOutput(io.Discard)
}

func TestNewController(t *testing.T) {
	kubeClient := fake.NewFakeClient()
	metrics := metrics.New(
		logrus.NewEntry(logrus.StandardLogger()),
		prometheus.NewRegistry(),
		kubeClient,
	)
	imageClient := &client.Client{}

	controller := NewPodReconciler(5*time.Minute, metrics, imageClient, kubeClient, testLogger, time.Hour, true)

	assert.NotNil(t, controller)
	assert.Equal(t, controller.defaultTestAll, true)
	assert.Equal(t, controller.Client, kubeClient)
	assert.NotNil(t, controller.VersionChecker)
}
func TestReconcile(t *testing.T) {
	imageClient := &client.Client{}

	tests := []struct {
		name            string
		pod             *corev1.Pod
		expectedError   bool
		expectedRequeue time.Duration
	}{
		{
			name: "Pod exists and is processed successfully",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
			},
			expectedError:   false,
			expectedRequeue: 5 * time.Minute,
		},
		{
			name:            "Pod does not exist",
			pod:             nil,
			expectedError:   false,
			expectedRequeue: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeClient := fake.NewFakeClient()
			metrics := metrics.New(
				logrus.NewEntry(logrus.StandardLogger()),
				prometheus.NewRegistry(),
				kubeClient,
			)

			controller := NewPodReconciler(5*time.Minute, metrics, imageClient, kubeClient, testLogger, 5*time.Minute, true)

			ctx := context.Background()

			if tt.pod != nil {
				err := kubeClient.Create(ctx, tt.pod)
				assert.NoError(t, err)
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-pod",
					Namespace: "default",
				},
			}

			result, err := controller.Reconcile(ctx, req)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedRequeue, result.RequeueAfter)
		})
	}
}

func TestSetupWithManager(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	metrics := metrics.New(
		logrus.NewEntry(logrus.StandardLogger()),
		prometheus.NewRegistry(),
		kubeClient,
	)
	imageClient := &client.Client{}
	controller := NewPodReconciler(5*time.Minute, metrics, imageClient, kubeClient, testLogger, time.Hour, true)

	mgr, err := manager.New(&rest.Config{}, manager.Options{LeaderElectionConfig: nil})
	require.NoError(t, err)

	err = controller.SetupWithManager(mgr)
	assert.NoError(t, err, "SetupWithManager should not return an error")
}
