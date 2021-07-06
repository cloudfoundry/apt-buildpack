package integration_test

import (
	"testing"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testFailure(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		app *cutlass.App
	)

	it.After(func() {
		app = DestroyApp(app)
	})

	context("as a final buildpack", func() {
		it.Before(func() {
			app = cutlass.New(settings.FixturePath)
			app.Buildpacks = []string{"https://github.com/cloudfoundry/binary-buildpack#master", "apt_buildpack"}
			app.SetEnv("BP_DEBUG", "1")
		})

		it("reports failure", func() {
			Expect(app.Push()).To(HaveOccurred())
			Eventually(app.Stdout.String, 3*time.Second).Should(MatchRegexp("(?i)failed"))

			Expect(app.Stdout.String()).To(ContainSubstring("Warning: the last buildpack is not compatible with multi-buildpack apps"))
		})
	})

	context("as a single buildpack", func() {
		it.Before(func() {
			app = cutlass.New(settings.FixturePath)
			app.Buildpacks = []string{"apt_buildpack"}
			app.SetEnv("BP_DEBUG", "1")
		})

		it("reports failure", func() {
			Expect(app.Push()).To(HaveOccurred())
			Eventually(app.Stdout.String, 3*time.Second).Should(MatchRegexp("(?i)failed"))

			Expect(app.Stdout.String()).To(ContainSubstring("Warning: this buildpack can only be run as a supply buildpack, it can not be run alone"))
		})
	})
}
