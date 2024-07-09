package selfhosted

import (
	"net/http"
	"regexp"
	"time"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/sirupsen/logrus"
)

type Options struct {
	Host      string
	Username  string
	Password  string
	Bearer    string
	TokenPath string
	Insecure  bool
	Timeout   int
	RetryMax  int
	CAPath    string
}

type Client struct {
	*http.Client
	*Options

	log *logrus.Entry

	hostRegex  *regexp.Regexp
	httpScheme string
}

type AuthResponse struct {
	Token string `json:"token"`
}

type TagResponse struct {
	Tags []string `json:"tags"`
}

type ManifestResponse struct {
	Digest       string
	Architecture api.Architecture `json:"architecture"`
	History      []History        `json:"history"`
}

type History struct {
	V1Compatibility string `json:"v1Compatibility"`
}

type V1Compatibility struct {
	Created time.Time `json:"created,omitempty"`
}
