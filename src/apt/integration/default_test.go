package integration_test

import (
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testDefault(platform switchblade.Platform, fixturePath string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect     = NewWithT(t).Expect
			Eventually = NewWithT(t).Eventually

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

		it("supplies apt packages to later buildpacks", func() {
			deployment, logs, err := platform.Deploy.
				WithBuildpacks("apt_buildpack", rubyBuildpackName, "binary_buildpack").
				WithEnv(map[string]string{"BP_DEBUG": "1"}).
				Execute(name, fixturePath)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainLines(ContainSubstring("Downloading apt packages")))
			Expect(logs).To(ContainLines(ContainSubstring("The following NEW packages will be installed:")))
			Expect(logs).To(ContainLines(ContainSubstring("bosh-cli cf-cli")))

			Expect(logs).To(ContainLines(ContainSubstring("Installing apt packages")))
			Expect(logs).NotTo(ContainLines(ContainSubstring("The following packages cannot be authenticated")))

			// installing packages from the default repo
			Eventually(deployment).Should(Serve(ContainSubstring("BOSH: version 2")).WithEndpoint("/bosh"))

			// installing packages from a specific file location
			Eventually(deployment).Should(Serve(ContainSubstring("Jq: jq-1")).WithEndpoint("/jq"))

			// installing a package from a specific repository with a lower priority
			Eventually(deployment).Should(Serve(ContainSubstring("cf version 6.38.0+7ddf0aadd.2018-08-07")).WithEndpoint("/cf"))
		})
	}
}
