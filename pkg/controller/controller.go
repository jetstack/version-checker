package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/clock"

	"github.com/jetstack/version-checker/pkg/client"
	"github.com/jetstack/version-checker/pkg/controller/scheduler"
	"github.com/jetstack/version-checker/pkg/metrics"
	"github.com/jetstack/version-checker/pkg/version"
)

const (
	numWorkers = 5
)

// controller is the main controller that check and exposes metrics on
// versions.
type Controller struct {
	log *logrus.Entry

	kubeClient         kubernetes.Interface
	podLister          corev1listers.PodLister
	workqueue          workqueue.RateLimitingInterface
	scheduledWorkQueue scheduler.ScheduledWorkQueue

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
	workqueue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	scheduledWorkQueue := scheduler.NewScheduledWorkQueue(clock.RealClock{}, workqueue.Add)

	c := &Controller{
		log:                log.WithField("module", "controller"),
		kubeClient:         kubeClient,
		workqueue:          workqueue,
		scheduledWorkQueue: scheduledWorkQueue,
		versionGetter:      version.New(log, imageClient, cacheTimeout),
		metrics:            metrics,
		cacheTimeout:       cacheTimeout,
		imageCache:         make(map[string]imageCacheItem),
		defaultTestAll:     defaultTestAll,
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
		AddFunc: func(obj interface{}) { c.addObject(obj) },
		UpdateFunc: func(old, new interface{}) {
			if key, err := cache.MetaNamespaceKeyFunc(old); err == nil {
				c.scheduledWorkQueue.Forget(key)
			}
			c.addObject(new)
		},
		DeleteFunc: func(obj interface{}) { c.addObject(obj) },
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

	// Start image tag garbage collector
	go c.versionGetter.StartGarbageCollector(c.cacheTimeout / 2)

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

		key, ok := obj.(string)
		if !ok {
			return
		}

		if err := c.processNextWorkItem(ctx, key); err != nil {
			c.log.Error(err.Error())
		}
	}
}

func (c *Controller) addObject(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	c.workqueue.AddRateLimited(key)
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem(ctx context.Context, key string) error {
	defer c.workqueue.Done(key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.log.Error(err, "invalid resource key")
		return nil
	}

	pod, err := c.podLister.Pods(namespace).Get(name)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		// If the pod has been deleted, remove from metrics
		for _, container := range pod.Spec.Containers {
			imageURL, currentTag, currentSHA := urlTagSHAFromImage(container.Image)

			c.log.Debugf("removing deleted container from metrics: %s/%s/%s: %s %s",
				pod.Namespace, pod.Name, container.Name, imageURL,
				metricsLabel(currentTag, currentSHA))

			c.metrics.RemoveImage(pod.Namespace, pod.Name, container.Name, imageURL)
		}

		return nil
	}

	if err := c.sync(ctx, pod); err != nil {
		c.scheduledWorkQueue.Add(pod, time.Second*20)
		return fmt.Errorf("error syncing '%s/%s': %s, requeuing",
			pod.Name, pod.Namespace, err)
	}

	// Check the image tag again after the cache timeout.
	c.scheduledWorkQueue.Add(key, c.cacheTimeout)

	return nil
}
