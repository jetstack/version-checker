package controller

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/jetstack/version-checker/pkg/client"
	"github.com/jetstack/version-checker/pkg/controller/checker"
	"github.com/jetstack/version-checker/pkg/controller/search"
	"github.com/jetstack/version-checker/pkg/metrics"
	"github.com/jetstack/version-checker/pkg/version"

	"github.com/sirupsen/logrus"
)

const (
	numWorkers = 10
)

type PodReconciler struct {
	k8sclient.Client
	Log             *logrus.Entry
	Metrics         *metrics.Metrics
	VersionChecker  *checker.Checker
	RequeueDuration time.Duration // Configurable reschedule duration

	defaultTestAll bool
}

func NewPodReconciler(
	cacheTimeout time.Duration,
	metrics *metrics.Metrics,
	imageClient *client.Client,
	kubeClient k8sclient.Client,
	log *logrus.Entry,
	requeueDuration time.Duration,
	defaultTestAll bool,
) *PodReconciler {
	log = log.WithField("controller", "pod")
	versionGetter := version.New(log, imageClient, cacheTimeout)
	search := search.New(log, cacheTimeout, versionGetter)

	c := &PodReconciler{
		Log:             log,
		Client:          kubeClient,
		Metrics:         metrics,
		VersionChecker:  checker.New(search),
		RequeueDuration: requeueDuration,
		defaultTestAll:  defaultTestAll,
	}

	return c
}

// Reconcile is triggered whenever a watched object changes.
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithField("pod", req.NamespacedName)

	// Fetch the Pod instance
	pod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, pod)
	if apierrors.IsNotFound(err) {
		// Pod deleted, remove from metrics
		log.Info("Pod not found, removing from metrics")
		r.Metrics.RemovePod(req.Namespace, req.Name)
		return ctrl.Result{Requeue: false}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	// Perform the version check (your sync logic)
	if err := r.sync(ctx, pod); err != nil {
		log.Error(err, "Failed to process pod")
		// Requeue after some time in case of failure
		return ctrl.Result{RequeueAfter: (r.RequeueDuration / 2)}, nil
	}

	// Schedule next check
	return ctrl.Result{RequeueAfter: r.RequeueDuration}, nil
}

// SetupWithManager initializes the controller
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	LeaderElect := false
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}, builder.OnlyMetadata).
		WithOptions(controller.Options{MaxConcurrentReconciles: numWorkers, NeedLeaderElection: &LeaderElect}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(_ event.TypedCreateEvent[k8sclient.Object]) bool { return true },
			UpdateFunc: func(_ event.TypedUpdateEvent[k8sclient.Object]) bool { return true },
			DeleteFunc: func(e event.TypedDeleteEvent[k8sclient.Object]) bool {
				r.Log.Infof("Pod deleted: %s/%s", e.Object.GetNamespace(), e.Object.GetName())
				r.Metrics.RemovePod(e.Object.GetNamespace(), e.Object.GetName())
				return false // Do not trigger reconciliation for deletes
			},
		}).
		Complete(r)
}
