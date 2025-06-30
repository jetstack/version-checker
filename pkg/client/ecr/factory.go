package ecr

import (
	"regexp"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/sirupsen/logrus"
)

var (
	ecrPattern = regexp.MustCompile(`(^[a-zA-Z0-9][a-zA-Z0-9-_]*)\.dkr\.ecr(\-fips)?\.([a-zA-Z0-9][a-zA-Z0-9-_]*)\.amazonaws\.com(\.cn)?$`)
)

var _ api.ImageClientFactory = (*Factory)(nil)
var _ authn.Keychain = (*Factory)(nil)

type Factory struct {
	opts Options
}

func NewFactory(opts Options) *Factory {
	return &Factory{opts: opts}
}

func (f *Factory) Name() string {
	return "ecr"
}

func (f *Factory) IsHost(host string) bool {
	return ecrPattern.MatchString(host)
}

func (f *Factory) NewClient(_ *authn.AuthConfig, log *logrus.Entry) (api.ImageClient, error) {
	return NewClient(f.opts, nil, log), nil
}

func (f *Factory) Resolve(res authn.Resource) (authn.Authenticator, error) {
	// We don't support ECR authentication via keychain
	// We'll use the default k8schain handler and ECR Helper
	return nil, nil
}
