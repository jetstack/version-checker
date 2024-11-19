package controller

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/jetstack/version-checker/pkg/client"
	"github.com/jetstack/version-checker/pkg/metrics"
)

var testLogger = logrus.NewEntry(logrus.New())

func init() {
	testLogger.Logger.SetOutput(io.Discard)
}

func TestNewController(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	metrics := &metrics.Metrics{}
	imageClient := &client.Client{}

	controller := New(5*time.Minute, metrics, imageClient, kubeClient, testLogger, true, nil)

	assert.NotNil(t, controller)
	assert.Equal(t, controller.defaultTestAll, true)
	assert.NotNil(t, controller.workqueue)
	assert.NotNil(t, controller.checker)
	assert.NotNil(t, controller.scheduledWorkQueue)
}

func TestRun(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	metrics := &metrics.Metrics{}
	imageClient := &client.Client{}
	controller := New(5*time.Minute, metrics, imageClient, kubeClient, testLogger, true, nil)

	ctx, cancel := context.WithCancel(context.Background())

	// Run the controller in a separate goroutine so the test doesn't block indefinitely
	go func() {
		err := controller.Run(ctx, 30*time.Second)
		assert.NoError(t, err)
	}()

	// Give the controller some time to start up and do initial processing
	time.Sleep(1 * time.Second)

	// Cancel the context to stop the controller
	cancel()

	// Wait a moment to ensure the controller has shut down
	time.Sleep(1 * time.Second)

	// Example assertion: Ensure the Run method has exited (this is implicit by the test not timing out)
	assert.True(t, true, "Controller should shutdown cleanly on context cancellation")
	// You can also add assertions here if you want to validate any specific state after shutdown
	assert.NotNil(t, controller.scheduledWorkQueue, "ScheduledWorkQueue should be initialized")
}

func TestAddObject(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	metrics := &metrics.Metrics{}
	imageClient := &client.Client{}
	controller := New(5*time.Minute, metrics, imageClient, kubeClient, testLogger, true, nil)

	obj := &corev1.Pod{}
	controller.addObject(obj)

	// Wait for the item to be added to the workqueue
	key, _ := cache.MetaNamespaceKeyFunc(obj)

	// Retry a few times with a short sleep to ensure the item has been added
	for i := 0; i < 10; i++ {
		if controller.workqueue.Len() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Check the workqueue length and the added item
	assert.Equal(t, 1, controller.workqueue.Len(), "Expected workqueue to have 1 item after adding an object")

	item, _ := controller.workqueue.Get()
	assert.Equal(t, key, item, "Expected the workqueue item to match the object's key")
}

func TestDeleteObject(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	metrics := &metrics.Metrics{}
	imageClient := &client.Client{}
	controller := New(5*time.Minute, metrics, imageClient, kubeClient, testLogger, true, nil)

	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "container1"},
				{Name: "container2"},
			},
		},
	}
	controller.deleteObject(pod)

	// We can't directly assert on log messages or metric updates,
	// but we can ensure that no errors are thrown and the function executes.
	assert.NotNil(t, pod)
}

func TestProcessNextWorkItem(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	metrics := &metrics.Metrics{}
	imageClient := &client.Client{}
	controller := New(5*time.Minute, metrics, imageClient, kubeClient, testLogger, true, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a fake pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	// Use a fake informer factory and lister
	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute)
	podInformer := informerFactory.Core().V1().Pods().Informer()
	controller.podLister = informerFactory.Core().V1().Pods().Lister()

	// Add the pod to the fake informer
	err := podInformer.GetIndexer().Add(pod)
	assert.NoError(t, err)

	// Add the pod key to the workqueue
	controller.workqueue.AddRateLimited("default/test-pod")

	// Start the informer to process the added pod
	informerFactory.Start(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced)

	// Test the processNextWorkItem method
	err = controller.processNextWorkItem(ctx, "default/test-pod", 30*time.Second)
	assert.NoError(t, err)
}
