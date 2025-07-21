package search

import (
	"context"
	"time"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/controller/search"
)

var _ search.Searcher = &FakeSearch{}

type FakeSearch struct {
	latestImageF     func() (*api.ImageTag, error)
	resolveSHAToTagF func() (string, error)
}

func New() *FakeSearch {
	return &FakeSearch{
		latestImageF: func() (*api.ImageTag, error) {
			return nil, nil
		},
		resolveSHAToTagF: func() (string, error) {
			return "", nil
		},
	}
}

func (f *FakeSearch) With(image *api.ImageTag, err error) *FakeSearch {
	f.latestImageF = func() (*api.ImageTag, error) {
		return image, err
	}
	return f
}

func (f *FakeSearch) LatestImage(context.Context, string, *api.Options) (*api.ImageTag, error) {
	return f.latestImageF()
}

func (f *FakeSearch) ResolveSHAToTag(ctx context.Context, imageURL string, imageSHA string) (string, error) {
	return f.resolveSHAToTagF()
}
func (f *FakeSearch) Run(time.Duration) {
}
