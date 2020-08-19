package controller

import "testing"

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
