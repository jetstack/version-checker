package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/controller/options"
	versionerrors "github.com/jetstack/version-checker/pkg/version/errors"
)

// sync will enqueue a given pod to run against the version checker.
func (c *PodReconciler) sync(ctx context.Context, pod *corev1.Pod) error {
	log := c.Log.WithFields(logrus.Fields{"name": pod.Name, "namespace": pod.Namespace})

	builder := options.New(pod.Annotations)

	var errs []string
	for _, container := range pod.Spec.InitContainers {
		if err := c.syncContainer(ctx, log, builder, pod, &container, "init"); err != nil {
			errs = append(errs, err.Error())
		}
	}
	for _, container := range pod.Spec.Containers {
		if err := c.syncContainer(ctx, log, builder, pod, &container, "container"); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to sync pod %s/%s: %s",
			pod.Namespace, pod.Name, strings.Join(errs, ","))
	}

	return nil
}

// syncContainer will enqueue a given container to check the version.
func (c *PodReconciler) syncContainer(ctx context.Context,
	log *logrus.Entry,
	builder *options.Builder,
	pod *corev1.Pod,
	container *corev1.Container,
	containerType string,
) error {
	// If not enabled, exit early
	if !builder.IsEnabled(c.defaultTestAll, container.Name) {
		c.Metrics.RemoveImage(pod.Namespace, pod.Name, container.Name, containerType)
		return nil
	}

	opts, err := builder.Options(container.Name)
	if err != nil {
		return fmt.Errorf("failed to build options from annotations for %q: %s",
			container.Name, err)
	}

	log = log.WithField("container", container.Name)
	log.Debug("processing container image")

	err = c.checkContainer(ctx, log, pod, container, containerType, opts)
	// Don't re-sync, if no version found meeting search criteria
	if versionerrors.IsNoVersionFound(err) {
		log.Error(err.Error())
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check container image %q: %s",
			container.Name, err)
	}

	return nil
}

// checkContainer will check the given container and options, and update
// metrics according to the result.
func (c *PodReconciler) checkContainer(ctx context.Context, log *logrus.Entry,
	pod *corev1.Pod,
	container *corev1.Container,
	containerType string,
	opts *api.Options,
) error {
	startTime := time.Now()
	defer func() {
		c.Metrics.RegisterImageDuration(pod.Namespace, pod.Name, container.Name, container.Image, startTime)
	}()

	result, err := c.VersionChecker.Container(ctx, log, pod, container, opts)
	if err != nil {
		// Report the error using ErrorsReporting
		c.Metrics.ReportError(pod.Namespace, pod.Name, container.Name, container.Image)
		return err
	}

	// If no result ready yet, exit early
	if result == nil {
		return nil
	}

	if result.IsLatest {
		log.Debugf("image is latest %s:%s",
			result.ImageURL, result.CurrentVersion)
	} else {
		log.Debugf("image is not latest %s: %s -> %s",
			result.ImageURL, result.CurrentVersion, result.LatestVersion)
	}

	c.Metrics.AddImage(pod.Namespace, pod.Name,
		container.Name, containerType,
		result.ImageURL, result.IsLatest,
		result.CurrentVersion, result.LatestVersion,
	)

	return nil
}
