package controller

import (
	"context"
	"fmt"
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
	"k8s.io/utils/clock"

	"github.com/jetstack/version-checker/pkg/client"
	"github.com/jetstack/version-checker/pkg/controller/checker"
	"github.com/jetstack/version-checker/pkg/controller/scheduler"
	"github.com/jetstack/version-checker/pkg/controller/search"
	"github.com/jetstack/version-checker/pkg/metrics"
	"github.com/jetstack/version-checker/pkg/version"
)

const (
	numWorkers = 10
)

// Controller is the main controller that check and exposes metrics on
// versions.
type Controller struct {
	log *logrus.Entry

	kubeClient         kubernetes.Interface
	podLister          corev1listers.PodLister
	workqueue          workqueue.TypedRateLimitingInterface[any]
	scheduledWorkQueue scheduler.ScheduledWorkQueue

	metrics *metrics.Metrics
	checker *checker.Checker

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
	workqueue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[any]())
	scheduledWorkQueue := scheduler.NewScheduledWorkQueue(clock.RealClock{}, workqueue.Add)

	log = log.WithField("module", "controller")
	versionGetter := version.New(log, imageClient, cacheTimeout)
	search := search.New(log, cacheTimeout, versionGetter)

	c := &Controller{
		log:                log,
		kubeClient:         kubeClient,
		workqueue:          workqueue,
		scheduledWorkQueue: scheduledWorkQueue,
		metrics:            metrics,
		checker:            checker.New(search),
		defaultTestAll:     defaultTestAll,
	}

	return c
}

// Run is a blocking func that will run the controller.
func (c *Controller) Run(ctx context.Context, cacheRefreshRate time.Duration) error {
	defer c.workqueue.ShutDown()

	sharedInformerFactory := informers.NewSharedInformerFactoryWithOptions(c.kubeClient, time.Second*30)
	c.podLister = sharedInformerFactory.Core().V1().Pods().Lister()
	podInformer := sharedInformerFactory.Core().V1().Pods().Informer()
	_, err := podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.addObject,
		UpdateFunc: func(old, new interface{}) {
			if key, err := cache.MetaNamespaceKeyFunc(old); err == nil {
				c.scheduledWorkQueue.Forget(key)
			}
			c.addObject(new)
		},
		DeleteFunc: c.deleteObject,
	})
	if err != nil {
		return fmt.Errorf("error creating podInformer: %s", err)
	}

	c.log.Info("starting control loop")
	sharedInformerFactory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced) {
		return fmt.Errorf("error waiting for informer caches to sync")
	}

	c.log.Info("starting workers")
	// Launch 10 workers to process pod resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(func() { c.runWorker(ctx, cacheRefreshRate) }, time.Second, ctx.Done())
	}

	// Start image tag garbage collector
	go c.checker.Search().Run(cacheRefreshRate)

	<-ctx.Done()

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker(ctx context.Context, searchReschedule time.Duration) {
	for {
		obj, shutdown := c.workqueue.Get()
		if shutdown {
			return
		}

		key, ok := obj.(string)
		if !ok {
			return
		}

		if err := c.processNextWorkItem(ctx, key, searchReschedule); err != nil {
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

func (c *Controller) deleteObject(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return
	}

	for _, container := range pod.Spec.Containers {
		c.log.WithFields(
			logrus.Fields{"pod": pod.Name, "container": container.Name, "namespace": pod.Namespace},
		).Debug("removing deleted pod containers from metrics")
		c.metrics.RemoveImage(pod.Namespace, pod.Name, container.Name, "init")
		c.metrics.RemoveImage(pod.Namespace, pod.Name, container.Name, "container")
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem(ctx context.Context, key string, searchReschedule time.Duration) error {
	defer c.workqueue.Done(key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.log.Error(err, "invalid resource key")
		return nil
	}

	pod, err := c.podLister.Pods(namespace).Get(name)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if err := c.sync(ctx, pod); err != nil {
		c.scheduledWorkQueue.Add(pod, time.Second*20)
		return fmt.Errorf("error syncing '%s/%s': %s, requeuing",
			pod.Name, pod.Namespace, err)
	}

	// Check the image tag again after the cache timeout.
	c.scheduledWorkQueue.Add(key, searchReschedule)

	return nil
}
