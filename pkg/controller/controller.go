package controller

import (
	//"time"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/masterminds/semver"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/joshvanl/version-checker/pkg/api"
	"github.com/joshvanl/version-checker/pkg/version"
)

type controller struct {
	log *logrus.Entry

	versionGetter *version.VersionGetter
}

func Run(ctx context.Context, kubeClient kubernetes.Interface) error {
	c := &controller{
		log:           logrus.NewEntry(logrus.New()),
		versionGetter: version.New(),
	}

	sharedInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, time.Second*30)
	podInformer := sharedInformerFactory.Core().V1().Pods().Informer()
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.enqueue(ctx, obj) },
		UpdateFunc: func(_, obj interface{}) { c.enqueue(ctx, obj) },
		// DeleteFunc // TODO: delete from metrics and cache
	})

	c.log.Info("starting control loop")
	sharedInformerFactory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced) {
		return fmt.Errorf("error waiting for informer caches to sync")
	}

	<-ctx.Done()

	return nil
}

// enqueue will enqueue a given pod to run against the version checker.
func (c *controller) enqueue(ctx context.Context, obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		c.log.Errorf("non-pod type passed to enqueue: %+v", obj)
		return
	}

	// TODO: add option to enable all pods
	//if enable, ok := pod.Annotations[api.EnableAnnotationKey]; !ok || enable != "true" {
	//	return
	//}

	log := c.log.WithField("name", pod.Name).WithField("namespace", pod.Namespace)

	log.Debug("processing pod images")

	for _, container := range pod.Spec.Containers {
		log = log.WithField("container", container.Name)

		opts, err := c.buildOptions(container.Name, pod.Annotations)
		if err != nil {
			log.Errorf("failed to build options from annotations for %q: %s",
				container.Name, err)
			return
		}

		if err := c.testContainerImage(ctx, log, container.Image, opts); err != nil {
			log.Errorf("failed to test container image: %s", err)
			continue
		}
	}
}

// testContainerImage will test a given image version to the latest image
// available in the remote registry given the options.
// TODO: create a cache
func (c *controller) testContainerImage(ctx context.Context, log *logrus.Entry, image string, opts *api.Options) error {
	imageSplit := strings.Split(image, ":")
	if len(imageSplit) != 2 {
		return fmt.Errorf("got unexpected image format [image:tag]: %s", image)
	}
	imageURL, tag := imageSplit[0], imageSplit[1]

	// TODO: handle SHA only use with full tag list
	tagV, err := semver.NewVersion(tag)
	if err != nil {
		return fmt.Errorf("failed to parse image tag: %s", err)
	}

	latestImage, _, err := c.versionGetter.LatestTagFromImage(ctx, opts, imageURL)
	if err != nil {
		return err
	}

	// TODO: handle SHA only

	if tagV.LessThan(latestImage.SemVer) {
		log.Infof("image is not latest %s: %s -> %s",
			imageURL, tag, latestImage.Tag)
	} else {
		log.Infof("image is latest %s:%s",
			imageURL, tag)
	}

	return nil
}

// buildOptions will build the tag options based on pod annotations.
func (c *controller) buildOptions(containerName string, annotations map[string]string) (*api.Options, error) {
	var (
		opts      api.Options
		errs      []string
		setNonSha bool
	)

	if useSHA, ok := annotations[api.UseSHAAnnotationKey+"/"+containerName]; ok && useSHA == "true" {
		opts.UseSHA = true
	}

	if usePreRelease, ok := annotations[api.UsePreReleaseAnnotationKey+"/"+containerName]; ok && usePreRelease == "true" {
		setNonSha = true
		opts.UsePreRelease = true
	}

	if matchRegex, ok := annotations[api.MatchRegexAnnotationKey+"/"+containerName]; ok {
		setNonSha = true

		regexMatcher, err := regexp.Compile(matchRegex)
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to compile regex at annotation %q: %s",
				api.MatchRegexAnnotationKey, err))
		} else {
			opts.RegexMatcher = regexMatcher
		}
	}

	if pinMajor, ok := annotations[api.PinMajorAnnotationKey+"/"+containerName]; ok {
		setNonSha = true

		ma, err := strconv.ParseInt(pinMajor, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to parse %s: %s",
				api.PinMajorAnnotationKey+"/"+containerName, err))
		} else {
			opts.PinMajor = &ma
		}
	}

	if pinMinor, ok := annotations[api.PinMinorAnnotationKey+"/"+containerName]; ok {
		setNonSha = true

		if opts.PinMajor == nil {
			errs = append(errs, fmt.Sprintf("unable to set %q without setting %q",
				api.PinMinorAnnotationKey+"/"+containerName, api.PinMajorAnnotationKey+"/"+containerName))
		} else {

			mi, err := strconv.ParseInt(pinMinor, 10, 64)
			if err != nil {
				errs = append(errs, fmt.Sprintf("failed to parse %s: %s",
					api.PinMinorAnnotationKey+"/"+containerName, err))
			} else {
				opts.PinMinor = &mi
			}
		}
	}

	if pinPatch, ok := annotations[api.PinPatchAnnotationKey+"/"+containerName]; ok {
		setNonSha = true

		if opts.PinMajor == nil && opts.PinMinor == nil {
			errs = append(errs, fmt.Sprintf("unable to set %q without setting %q or %q",
				api.PinPatchAnnotationKey+"/"+containerName,
				api.PinMinorAnnotationKey+"/"+containerName,
				api.PinMajorAnnotationKey+"/"+containerName))
		} else {

			pa, err := strconv.ParseInt(pinPatch, 10, 64)
			if err != nil {
				errs = append(errs, fmt.Sprintf("failed to parse %s: %s",
					api.PinPatchAnnotationKey+"/"+containerName, err))
			} else {
				opts.PinPatch = &pa
			}
		}
	}

	if opts.UseSHA && setNonSha {
		errs = append(errs, fmt.Sprintf("cannot define %q with any semver otions",
			api.UseSHAAnnotationKey+"/"+containerName))
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to build version options: %s",
			strings.Join(errs, ", "))
	}

	return &opts, nil
}
