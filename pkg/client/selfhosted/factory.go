package selfhosted

import (
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/sirupsen/logrus"
)

var _ api.ImageClientFactory = (*Factory)(nil)

type Options struct {
	Host        string
	Username    string
	Password    string
	Bearer      string
	TokenPath   string
	Insecure    bool
	CAPath      string
	Transporter http.RoundTripper
}

type Factory struct {
	opts Options
}

func NewFactory(opts Options) api.ImageClientFactory {
	return &Factory{opts: opts}

}

// We don't have a self-hosted image client, so we return nil for all methods.
func (f *Factory) IsHost(_ string) bool {
	return false
}

func (f *Factory) Name() string {
	return "selfhosted"
}

func (f *Factory) NewClient(_ *authn.AuthConfig, logger *logrus.Entry) (api.ImageClient, error) {
	return nil, nil
}

func (f *Factory) Resolve(image authn.Resource) (authn.Authenticator, error) {
	return authn.FromConfig(authn.AuthConfig{
		Username: "",
		Password: ""},
	), nil
}
