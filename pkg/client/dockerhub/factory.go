package dockerhub

import (
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/sirupsen/logrus"
)

var _ api.ImageClientFactory = (*Factory)(nil)
var _ authn.Keychain = (*Factory)(nil)

var (
	dockerReg = regexp.MustCompile(`(^(.*\.)?docker.com$)|(^(.*\.)?docker.io$)`)
)

type Factory struct {
	opts Options
}

func NewFactory(opts Options) *Factory {
	return &Factory{opts: opts}
}

func (f *Factory) NewClient(auth *authn.AuthConfig, log *logrus.Entry) (api.ImageClient, error) {
	return NewClient(f.opts, auth, log)
}

func (f *Factory) Name() string {
	return "dockerhub"
}

func (f *Factory) IsHost(host string) bool {
	return host == "" || dockerReg.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	split := strings.Split(path, "/")

	lenSplit := len(split)
	if lenSplit == 1 {
		return "library", split[0]
	}

	return split[lenSplit-2], split[lenSplit-1]
}

func (f *Factory) Resolve(res authn.Resource) (authn.Authenticator, error) {
	if !f.IsHost(res.RegistryStr()) {
		return authn.Anonymous, nil
	}
	return authn.FromConfig(authn.AuthConfig{
		Username:      f.opts.Username,
		Password:      f.opts.Password,
		RegistryToken: f.opts.Token,
	}), nil
}
