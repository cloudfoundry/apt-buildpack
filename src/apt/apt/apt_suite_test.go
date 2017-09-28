package apt_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestApt(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Apt Suite")
}
