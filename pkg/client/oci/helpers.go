package oci

import (
	"strings"
	"time"
)

const (
	CreatedTimeAnnotation = "org.opencontainers.image.created"
	BuildDateAnnotation   = "org.label-schema.build-date"
)

func discoverTimestamp(annotations map[string]string) (timestamp time.Time, err error) {
	if t, ok := annotations[CreatedTimeAnnotation]; ok {
		timestamp, err = time.Parse(time.RFC3339,
			strings.Replace(t, " ", "T", 1),
		)
	} else if t, ok = annotations[BuildDateAnnotation]; ok {
		timestamp, err = time.Parse(time.RFC3339,
			strings.Replace(t, " ", "T", 1),
		)
	}

	return timestamp, err
}
