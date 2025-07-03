package keychains

import (
	"log"

	"github.com/sirupsen/logrus"

	"github.com/patrickmn/go-cache"

	cntreglog "github.com/google/go-containerregistry/pkg/logs"
	"k8s.io/client-go/kubernetes"
)

// New initializes a new keychain manager with given ttl and cleanup interval.
func New(logger *logrus.Entry, clientset kubernetes.Interface, opts *ManagerOpts) (obj Manager) {
	cache := cache.New(opts.CachingTTL, opts.CachingTTL*2)

	// We need to ensure that we set this for Pod and ServiceAccountKeychains
	cntreglog.Warn = log.New(logger.WriterLevel(logrus.WarnLevel), "", 0)
	cntreglog.Debug = log.New(logger.WriterLevel(logrus.DebugLevel), "", 0)

	switch opts.Mode {
	default:
	case ManualMode:
		obj = &ManualKeychain{}
	case PodMode:
		obj = &PodKeychain{client: clientset, log: logger, opts: opts, cache: cache}
	case ServiceAccountMode:
		obj = &ServiceAccountKeychain{client: clientset, log: logger, opts: opts, cache: cache}
	}

	return obj
}
