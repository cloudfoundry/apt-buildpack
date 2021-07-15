package integration_test

import (
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testFailure(platform switchblade.Platform, fixturePath string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect = NewWithT(t).Expect

			name string
		)

		it.Before(func() {
			var err error
			name, err = switchblade.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
		})

		context("as a final buildpack", func() {
			it("reports failure", func() {
				_, logs, err := platform.Deploy.
					WithBuildpacks("binary_buildpack", "apt_buildpack").
					Execute(name, fixturePath)
				Expect(err).To(MatchError(ContainSubstring("App staging failed")))
				Expect(logs).To(ContainLines(ContainSubstring("Warning: the last buildpack is not compatible with multi-buildpack apps")))
			})
		})

		context("as a single buildpack", func() {
			it("reports failure", func() {
				_, logs, err := platform.Deploy.
					WithBuildpacks("apt_buildpack").
					Execute(name, fixturePath)
				Expect(err).To(MatchError(ContainSubstring("App staging failed")))
				Expect(logs).To(ContainSubstring("Warning: this buildpack can only be run as a supply buildpack, it can not be run alone"))
			})
		})
	}
}
