package e2e

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	_ "github.com/jetstack/version-checker/test/e2e/suite"
)

func TestVersionChecker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VersionChecker Suite")
}
