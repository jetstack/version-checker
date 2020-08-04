package semver

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	versionRegex = regexp.MustCompile(`^v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?(.*)$`)
)

// SemVer is a struct to contain a SemVer of an image tag.
type SemVer struct {
	// version is the version number of a tag. 'Left', or smaller index, the
	// higher weight.
	version [3]int64

	// metadata holds the metadata, which is the string suffixed from the patch
	metadata string
}

func Parse(tag string) *SemVer {
	s := &SemVer{
		version: [3]int64{},
	}

	match := versionRegex.FindStringSubmatch(tag)
	if len(match) == 0 {
		s.metadata = tag
		return s
	}

	for i := 0; i < 3; i++ {
		if len(match[i+1]) > 0 {
			s.version[i], _ = strconv.ParseInt(strings.TrimPrefix(match[i+1], "."), 10, 64)
		}
	}
	s.metadata = match[4]

	return s
}

// LessThan will return true if the given semver is equal, or larger that the
// calling semver. If the calling SemVer has metadata, then ASCII comparison
// will take place on the version.
// e.g. v1.0.1-alpha.1 < v1.0.1-beta.0
func (s *SemVer) LessThan(other *SemVer) bool {
	// if s doesn't have metadata but other doest, false.
	if !s.HasMetaData() && other.HasMetaData() {
		return false
	}

	for i := 0; i < 3; i++ {
		if s.version[i] != other.version[i] {
			return s.version[i] < other.version[i]
		}
	}

	sparts := strings.Split(s.metadata, ".")
	oparts := strings.Split(other.metadata, ".")

	l := len(sparts)
	if len(oparts) > l {
		l = len(oparts)
	}

	for i := 0; i < l; i++ {
		if i > len(sparts) {
			return false
		}
		if i > len(oparts) {
			return true
		}

		if sparts[i] == oparts[i] {
			continue
		}

		si, se := strconv.ParseInt(sparts[i], 10, 64)
		oi, oe := strconv.ParseInt(oparts[i], 10, 64)

		// The case where both are strings compare the strings
		if se != nil && oe != nil {
			return sparts[i] < oparts[i]
		} else if se != nil {
			// s not a number
			return false
		} else if oe != nil {
			// other not a number
			return true
		}

		return si < oi
	}

	return false
}

// HasMetaData returns whether this SemVer has metadata. MetaData is defined
// as a tag containing anything after the patch digit.
// e.g. v1.0.1-gke.3, v1.0.1-alpha.0, v1.2.3.4
func (s *SemVer) HasMetaData() bool {
	return len(s.metadata) > 0
}

// Major returns the major version of this SemVer.
func (s *SemVer) Major() int64 {
	return s.version[0]
}

// Minor returns the minor version of this SemVer.
func (s *SemVer) Minor() int64 {
	return s.version[1]
}

// Patch returns the patch version of this SemVer.
func (s *SemVer) Patch() int64 {
	return s.version[2]
}
