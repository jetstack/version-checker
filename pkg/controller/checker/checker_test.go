package checker

import (
	"context"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/controller/internal/fake/search"
	"github.com/jetstack/version-checker/pkg/version/semver"
)

func TestContainer(t *testing.T) {
	tests := map[string]struct {
		statusSHA  string
		imageURL   string
		opts       *api.Options
		searchResp *api.ImageTag
		expResult  *Result
	}{
		"no status sha should return nil, nil": {
			statusSHA:  "",
			imageURL:   "version-checker:v0.2.0",
			opts:       nil,
			searchResp: nil,
			expResult:  nil,
		},
		"if v0.2.0 is latest version, but different sha, then not latest": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/version-checker:v0.2.0",
			opts:      new(api.Options),
			searchResp: &api.ImageTag{
				Tag: "v0.2.0",
				SHA: "sha:456",
			},
			expResult: &Result{
				CurrentVersion: "v0.2.0@sha:123",
				LatestVersion:  "v0.2.0@sha:456",
				ImageURL:       "localhost:5000/version-checker",
				IsLatest:       false,
			},
		},
		"if v0.2.0 is latest version, but same sha, then latest": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/version-checker:v0.2.0",
			opts:      new(api.Options),
			searchResp: &api.ImageTag{
				Tag: "v0.2.0",
				SHA: "sha:123",
			},
			expResult: &Result{
				CurrentVersion: "v0.2.0",
				LatestVersion:  "v0.2.0",
				ImageURL:       "localhost:5000/version-checker",
				IsLatest:       true,
			},
		},
		"if v0.2.0@sha:123 is wrong sha, then not latest": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/version-checker:v0.2.0@sha:123",
			opts:      new(api.Options),
			searchResp: &api.ImageTag{
				Tag: "v0.2.0",
				SHA: "sha:456",
			},
			expResult: &Result{
				CurrentVersion: "v0.2.0@sha:123",
				LatestVersion:  "v0.2.0@sha:456",
				ImageURL:       "localhost:5000/version-checker",
				IsLatest:       false,
			},
		},
		"if v0.2.0@sha:123 is correct sha, then latest": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/version-checker:v0.2.0@sha:123",
			opts:      new(api.Options),
			searchResp: &api.ImageTag{
				Tag: "v0.2.0",
				SHA: "sha:123",
			},
			expResult: &Result{
				CurrentVersion: "v0.2.0@sha:123",
				LatestVersion:  "v0.2.0@sha:123",
				ImageURL:       "localhost:5000/version-checker",
				IsLatest:       true,
			},
		},
		"if empty is not latest version, then return false": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/version-checker",
			opts:      new(api.Options),
			searchResp: &api.ImageTag{
				Tag: "v0.2.0",
				SHA: "sha:456",
			},
			expResult: &Result{
				CurrentVersion: "sha:123",
				LatestVersion:  "v0.2.0@sha:456",
				ImageURL:       "localhost:5000/version-checker",
				IsLatest:       false,
			},
		},
		"if empty is latest version, then return true": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/version-checker",
			opts:      new(api.Options),
			searchResp: &api.ImageTag{
				Tag: "",
				SHA: "sha:123",
			},
			expResult: &Result{
				CurrentVersion: "sha:123",
				LatestVersion:  "sha:123",
				ImageURL:       "localhost:5000/version-checker",
				IsLatest:       true,
			},
		},
		"if latest is not latest version, then return false": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/version-checker:latest",
			opts:      new(api.Options),
			searchResp: &api.ImageTag{
				Tag: "v0.2.0",
				SHA: "sha:456",
			},
			expResult: &Result{
				CurrentVersion: "sha:123",
				LatestVersion:  "v0.2.0@sha:456",
				ImageURL:       "localhost:5000/version-checker",
				IsLatest:       false,
			},
		},
		"if latest is latest version, then return true": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/version-checker:latest",
			opts:      new(api.Options),
			searchResp: &api.ImageTag{
				Tag: "",
				SHA: "sha:123",
			},
			expResult: &Result{
				CurrentVersion: "sha:123",
				LatestVersion:  "sha:123",
				ImageURL:       "localhost:5000/version-checker",
				IsLatest:       true,
			},
		},
		"if latest is latest version, and OverrideURL is set, then return true": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/version-checker:latest",
			opts: &api.Options{
				OverrideURL: stringp("quay.io/jetstack/version-checker"),
			},
			searchResp: &api.ImageTag{
				Tag: "",
				SHA: "sha:123",
			},
			expResult: &Result{
				CurrentVersion: "sha:123",
				LatestVersion:  "sha:123",
				ImageURL:       "quay.io/jetstack/version-checker",
				IsLatest:       true,
			},
		},
		"if using v0.2.0 with use sha, but not latest, return false": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/version-checker:v0.2.0",
			opts: &api.Options{
				UseSHA: true,
			},
			searchResp: &api.ImageTag{
				Tag: "v0.2.0",
				SHA: "sha:456",
			},
			expResult: &Result{
				CurrentVersion: "v0.2.0@sha:123",
				LatestVersion:  "v0.2.0@sha:456",
				ImageURL:       "localhost:5000/version-checker",
				IsLatest:       false,
			},
		},
		"if using v0.2.0 with use sha, but latest, return true": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/version-checker:v0.2.0",
			opts: &api.Options{
				UseSHA: true,
			},
			searchResp: &api.ImageTag{
				Tag: "v0.2.0",
				SHA: "sha:123",
			},
			expResult: &Result{
				CurrentVersion: "v0.2.0@sha:123",
				LatestVersion:  "v0.2.0@sha:123",
				ImageURL:       "localhost:5000/version-checker",
				IsLatest:       true,
			},
		},
		"if using sha but not latest, return false": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/joshvanl/version-checker@sha:123",
			opts:      new(api.Options),
			searchResp: &api.ImageTag{
				Tag: "v0.2.0",
				SHA: "sha:456",
			},
			expResult: &Result{
				CurrentVersion: "sha:123",
				LatestVersion:  "v0.2.0@sha:456",
				ImageURL:       "localhost:5000/joshvanl/version-checker",
				IsLatest:       false,
			},
		},
		"if using sha but sha not latest, return false and no tag if non exists": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/joshvanl/version-checker@sha:123",
			opts:      new(api.Options),
			searchResp: &api.ImageTag{
				Tag: "",
				SHA: "sha:456",
			},
			expResult: &Result{
				CurrentVersion: "sha:123",
				LatestVersion:  "sha:456",
				ImageURL:       "localhost:5000/joshvanl/version-checker",
				IsLatest:       false,
			},
		},
		"if using sha and is latest, return true": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/joshvanl/version-checker@sha:123",
			opts:      new(api.Options),
			searchResp: &api.ImageTag{
				Tag: "v0.2.0",
				SHA: "sha:123",
			},
			expResult: &Result{
				CurrentVersion: "sha:123",
				LatestVersion:  "v0.2.0@sha:123",
				ImageURL:       "localhost:5000/joshvanl/version-checker",
				IsLatest:       true,
			},
		},
		"if using sha and is latest, return true and no tag if non exists": {
			statusSHA: "localhost:5000/version-checker@sha:123",
			imageURL:  "localhost:5000/joshvanl/version-checker@sha:123",
			opts:      new(api.Options),
			searchResp: &api.ImageTag{
				Tag: "",
				SHA: "sha:123",
			},
			expResult: &Result{
				CurrentVersion: "sha:123",
				LatestVersion:  "sha:123",
				ImageURL:       "localhost:5000/joshvanl/version-checker",
				IsLatest:       true,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			checker := New(search.New().With(test.searchResp, nil))
			pod := &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:    "test-name",
							ImageID: test.statusSHA,
						},
					},
				},
			}
			container := &corev1.Container{
				Name:  "test-name",
				Image: test.imageURL,
			}

			result, err := checker.Container(context.TODO(), logrus.NewEntry(logrus.New()), pod, container, test.opts)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}

			if !reflect.DeepEqual(test.expResult, result) {
				t.Errorf("got unexpected result, exp=%#+v got=%#+v",
					test.expResult, result)
			}
		})
	}
}

func TestContainerStatusImageSHA(t *testing.T) {
	tests := map[string]struct {
		status []corev1.ContainerStatus
		name   string
		expSHA string
	}{
		"if no status, then return ''": {
			status: []corev1.ContainerStatus{},
			name:   "test-name",
			expSHA: "",
		},
		"if status with wrong name, then return ''": {
			status: []corev1.ContainerStatus{
				{
					Name:    "foo",
					ImageID: "123",
				},
			},
			name:   "test-name",
			expSHA: "",
		},
		"if status with wrong name and correct, then return '456'": {
			status: []corev1.ContainerStatus{
				{
					Name:    "foo",
					ImageID: "123",
				},
				{
					Name:    "test-name",
					ImageID: "456",
				},
			},
			name:   "test-name",
			expSHA: "456",
		},
		"if status with multiple status, then return first '456'": {
			status: []corev1.ContainerStatus{
				{
					Name:    "foo",
					ImageID: "123",
				},
				{
					Name:    "test-name",
					ImageID: "456",
				},
				{
					Name:    "test-name",
					ImageID: "789",
				},
			},
			name:   "test-name",
			expSHA: "456",
		},
		"if status with includes URL, then return just SHA": {
			status: []corev1.ContainerStatus{
				{
					Name:    "foo",
					ImageID: "123",
				},
				{
					Name:    "test-name",
					ImageID: "localhost:5000/joshvanl/version-checker@sha:456",
				},
			},
			name:   "test-name",
			expSHA: "sha:456",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			pod := &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: test.status,
				},
			}

			if sha := containerStatusImageSHA(pod, test.name); sha != test.expSHA {
				t.Errorf("unexpected image status sha, exp=%s got=%s",
					test.expSHA, sha)
			}
		})
	}
}

func TestIsLatestOrEmptyTag(t *testing.T) {
	tests := map[string]struct {
		tag   string
		expIs bool
	}{
		"if empty, true": {
			tag:   "",
			expIs: true,
		},
		"if 'latest', true": {
			tag:   "latest",
			expIs: true,
		},
		"if anything, false": {
			tag:   "anything",
			expIs: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			checker := New(search.New())
			if is := checker.isLatestOrEmptyTag(test.tag); is != test.expIs {
				t.Errorf("unexpected isLatestOrEmptyTag exp=%t got=%t",
					test.expIs, is)
			}
		})
	}
}

func TestIsLatestSemver(t *testing.T) {
	tests := map[string]struct {
		imageURL, currentSHA string
		currentImage         *semver.SemVer
		searchResp           *api.ImageTag
		expLatestImage       *api.ImageTag
		expIsLatest          bool
	}{
		"if current semver is less, then is less": {
			imageURL:     "docker.io",
			currentSHA:   "123",
			currentImage: semver.Parse("v1.2.3"),
			searchResp: &api.ImageTag{
				Tag: "v1.2.4",
				SHA: "456",
			},
			expLatestImage: &api.ImageTag{
				Tag: "v1.2.4",
				SHA: "456",
			},
			expIsLatest: false,
		},
		"if current semver is equal, but semver missmatch, then false": {
			imageURL:     "docker.io",
			currentSHA:   "123",
			currentImage: semver.Parse("v1.2.4"),
			searchResp: &api.ImageTag{
				Tag: "v1.2.4",
				SHA: "456",
			},
			expLatestImage: &api.ImageTag{
				Tag: "v1.2.4@456",
				SHA: "456",
			},
			expIsLatest: false,
		},
		"if current semver is equal, and semver match, then true": {
			imageURL:     "docker.io",
			currentSHA:   "456",
			currentImage: semver.Parse("v1.2.4"),
			searchResp: &api.ImageTag{
				Tag: "v1.2.4",
				SHA: "456",
			},
			expLatestImage: &api.ImageTag{
				Tag: "v1.2.4",
				SHA: "456",
			},
			expIsLatest: true,
		},
		"if current semver is more, then true": {
			imageURL:     "docker.io",
			currentSHA:   "123",
			currentImage: semver.Parse("v1.2.5"),
			searchResp: &api.ImageTag{
				Tag: "v1.2.4",
				SHA: "456",
			},
			expLatestImage: &api.ImageTag{
				Tag: "v1.2.4",
				SHA: "456",
			},
			expIsLatest: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			checker := New(search.New().With(test.searchResp, nil))
			latestImage, isLatest, err := checker.isLatestSemver(context.TODO(), test.imageURL, test.currentSHA, test.currentImage, nil)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(latestImage, test.expLatestImage) {
				t.Errorf("got unexpected latest image, exp=%v got=%v",
					test.expLatestImage, latestImage)
			}

			if isLatest != test.expIsLatest {
				t.Errorf("got unexpected is latest image, exp=%t got=%t",
					test.expIsLatest, isLatest)
			}
		})
	}
}

func TestIsLatestSHA(t *testing.T) {
	tests := map[string]struct {
		imageURL, currentSHA string
		searchResp           *api.ImageTag
		expResult            *Result
	}{
		"if SHA not eqaual, then should be not equal": {
			imageURL:   "docker.io",
			currentSHA: "123",
			searchResp: &api.ImageTag{
				SHA: "456",
			},
			expResult: &Result{
				CurrentVersion: "123",
				LatestVersion:  "456",
				IsLatest:       false,
				ImageURL:       "docker.io",
			},
		},
		"if SHA eqaual, then should be equal": {
			imageURL:   "docker.io",
			currentSHA: "123",
			searchResp: &api.ImageTag{
				SHA: "123",
			},
			expResult: &Result{
				CurrentVersion: "123",
				LatestVersion:  "123",
				IsLatest:       true,
				ImageURL:       "docker.io",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			checker := New(search.New().With(test.searchResp, nil))
			result, err := checker.isLatestSHA(context.TODO(), test.imageURL, test.currentSHA, nil)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(result, test.expResult) {
				t.Errorf("got unexpected result, exp=%v got=%v",
					test.expResult, result)
			}
		})
	}
}

func TestURLAndTagFromImage(t *testing.T) {
	tests := map[string]struct {
		image             string
		url, version, sha string
	}{
		"no version or sha, return just image": {
			image:   "nginx",
			url:     "nginx",
			version: "",
			sha:     "",
		},
		"version, return image and version": {
			image:   "nginx:v1.0.0",
			url:     "nginx",
			version: "v1.0.0",
			sha:     "",
		},
		"sha, return image and sha": {
			image:   "nginx@sha:123",
			url:     "nginx",
			version: "",
			sha:     "sha:123",
		},
		"version and sha, return image, version, and sha": {
			image:   "nginx:v1.0.0@sha:123",
			url:     "nginx",
			version: "v1.0.0",
			sha:     "sha:123",
		},

		"url in image, return tag": {
			image:   "localhost:5000/version-checker:v0.2.0",
			url:     "localhost:5000/version-checker",
			version: "v0.2.0",
			sha:     "",
		},

		"url with port but no version": {
			image:   "localhost:5000/version-checker",
			url:     "localhost:5000/version-checker",
			version: "",
			sha:     "",
		},

		"url in image, return sha": {
			image:   "localhost:5000/version-checker@sha:123",
			url:     "localhost:5000/version-checker",
			version: "",
			sha:     "sha:123",
		},

		"url in image with version, return sha and version": {
			image:   "localhost:5000/version-checker:v0.1@sha:123",
			url:     "localhost:5000/version-checker",
			version: "v0.1",
			sha:     "sha:123",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			url, version, sha := urlTagSHAFromImage(test.image)
			if url != test.url || version != test.version || sha != test.sha {
				t.Errorf("unexpected response, exp=%q,%q,%q got=%q,%q,%q",
					test.url, test.version, test.sha,
					url, version, sha)
			}
		})
	}
}

func stringp(s string) *string {
	return &s
}
