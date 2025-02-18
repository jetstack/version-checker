package search

import (
	"context"
	"time"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/controller/search"
)

var _ search.Searcher = &FakeSearch{}

type FakeSearch struct {
	imageF       func() (*api.ImageTag, error)
	latestImageF func() (*api.ImageTag, error)
}

func New() *FakeSearch {
	return &FakeSearch{
		imageF: func() (*api.ImageTag, error) {
			return nil, nil
		},
		latestImageF: func() (*api.ImageTag, error) {
			return nil, nil
		},
	}
}

func (f *FakeSearch) WithImage(image *api.ImageTag, err error) *FakeSearch {
	f.imageF = func() (*api.ImageTag, error) {
		return image, err
	}
	return f
}

func (f *FakeSearch) Image(context.Context, string, string, string, bool) (*api.ImageTag, error) {
	return f.imageF()
}

func (f *FakeSearch) WithLatestImage(image *api.ImageTag, err error) *FakeSearch {
	f.latestImageF = func() (*api.ImageTag, error) {
		return image, err
	}
	return f
}

func (f *FakeSearch) LatestImage(context.Context, string, *api.Options) (*api.ImageTag, error) {
	return f.latestImageF()
}

func (f *FakeSearch) Run(time.Duration) {
}
