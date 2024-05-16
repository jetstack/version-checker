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
	metadata string
	original string
	version  [3]int64
}

func Parse(tag string) *SemVer {
	s := &SemVer{
		original: tag,
		version:  [3]int64{},
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
	if len(other.original) == 0 || len(s.original) == 0 {
		return len(s.original) < len(other.original)
	}

	// if s doesn't have metadata but other doesn't, false.
	if !s.HasMetaData() && other.HasMetaData() {
		return false
	}

	for i := 0; i < 3; i++ {
		if s.version[i] != other.version[i] {
			return s.version[i] < other.version[i]
		}
	}

	sWords := parseStringToWords(s.metadata)
	otherWords := parseStringToWords(other.metadata)
	l := len(sWords)
	if len(otherWords) > l {
		l = len(otherWords)
	}

	for i := 0; i < l; i++ {
		if i > len(sWords)-1 {
			return false
		}
		if i > len(otherWords)-1 {
			return true
		}

		if sWords[i].equal(otherWords[i]) {
			continue
		}

		return sWords[i].lessThan(otherWords[i])
	}

	return false
}

// Equal will return true if the given semver is equal.
func (s *SemVer) Equal(other *SemVer) bool {
	return s.original == other.original
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

func (s *SemVer) String() string {
	return s.original
}
