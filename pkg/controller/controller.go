package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/joshvanl/version-checker/pkg/client"
	"github.com/joshvanl/version-checker/pkg/metrics"
	"github.com/joshvanl/version-checker/pkg/version"
)

const (
	numWorkers = 5
)

// controller is the main controller that check and exposes metrics on
// versions.
type Controller struct {
	log *logrus.Entry

	kubeClient kubernetes.Interface
	podLister  corev1listers.PodLister
	workqueue  workqueue.RateLimitingInterface

	versionGetter *version.VersionGetter
	metrics       *metrics.Metrics

	cacheMu      sync.RWMutex
	cacheTimeout time.Duration
	imageCache   map[string]imageCacheItem

	defaultTestAll bool
}

func New(
	cacheTimeout time.Duration,
	metrics *metrics.Metrics,
	imageClient *client.Client,
	kubeClient kubernetes.Interface,
	log *logrus.Entry,
	defaultTestAll bool,
) *Controller {
	c := &Controller{
		log:            log.WithField("module", "controller"),
		kubeClient:     kubeClient,
		workqueue:      workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		versionGetter:  version.New(log, imageClient, cacheTimeout),
		metrics:        metrics,
		cacheTimeout:   cacheTimeout,
		imageCache:     make(map[string]imageCacheItem),
		defaultTestAll: defaultTestAll,
	}

	return c
}

// Run is a blocking func that will create and run new controller.
func (c *Controller) Run(ctx context.Context) error {
	defer c.workqueue.ShutDown()

	sharedInformerFactory := informers.NewSharedInformerFactoryWithOptions(c.kubeClient, time.Second*30)
	c.podLister = sharedInformerFactory.Core().V1().Pods().Lister()
	podInformer := sharedInformerFactory.Core().V1().Pods().Informer()
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.workqueue.Add(obj) },
		UpdateFunc: func(_, obj interface{}) { c.workqueue.Add(obj) },
		DeleteFunc: func(obj interface{}) { c.workqueue.Add(obj) },
	})

	c.log.Info("starting control loop")
	sharedInformerFactory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced) {
		return fmt.Errorf("error waiting for informer caches to sync")
	}

	c.log.Info("starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(func() { c.runWorker(ctx) }, time.Second, ctx.Done())
	}

	go c.garbageCollect(c.cacheTimeout / 2)

	<-ctx.Done()

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker(ctx context.Context) {
	for {
		obj, shutdown := c.workqueue.Get()
		if shutdown {
			return
		}

		if err := c.processNextWorkItem(ctx, obj); err != nil {
			c.log.Error(err)
		}
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem(ctx context.Context, obj interface{}) error {
	defer c.workqueue.Done(obj)

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		c.log.Errorf("non-pod type passed to sync: %+v", obj)
		c.workqueue.Forget(obj)
		return nil
	}

	if _, err := c.podLister.Pods(pod.Namespace).Get(pod.Name); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		// If the pod has been deleted, remove from metrics
		for _, container := range pod.Spec.Containers {
			imageURL, currentTag := urlAndTagFromImage(container.Image)

			c.log.Debugf("removing deleted container from metrics: %s/%s/%s: %s:%s",
				pod.Namespace, pod.Name, container.Name, imageURL, currentTag)
			c.metrics.RemoveImage(pod.Namespace, pod.Name, container.Name, imageURL, currentTag)
		}

		return nil
	}

	if err := c.sync(ctx, pod); err != nil {
		c.workqueue.AddAfter(pod, time.Second*20)
		return fmt.Errorf("error syncing '%s/%s': %s, requeuing",
			pod.Name, pod.Namespace, err)
	}

	c.workqueue.Forget(obj)
	return nil
}
