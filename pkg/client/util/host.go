package util

import (
	"fmt"
	"regexp"
)

// hostRe is a bare DNS name or bracketed IP literal with an optional port. It
// excludes every URL metacharacter (@ # ? / \ whitespace), so the host a
// check sees is the host a request is sent to.
var hostRe = regexp.MustCompile(`^(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*|\[[0-9A-Fa-f:.]+\])(?::[0-9]{1,5})?$`)

// ValidHost returns an error unless host is a bare hostname or IP literal with
// an optional port. Image hosts are pod-controlled and end up in request URLs.
func ValidHost(host string) error {
	if len(host) > 255 || !hostRe.MatchString(host) {
		return fmt.Errorf("invalid registry host %.64q", host)
	}
	return nil
}
