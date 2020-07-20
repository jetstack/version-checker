package util

import (
	"fmt"
	"strings"
)

func ParseSHADigest(sha string) (string, error) {
	shaSplit := strings.Split(sha, ":")
	if len(shaSplit) != 2 {
		return "", fmt.Errorf("got unexpected sha format ([hash function:...): %s", sha)
	}

	return shaSplit[1], nil
}
