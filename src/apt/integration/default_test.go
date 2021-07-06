package integration_test

import (
	"testing"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDefault(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		app *cutlass.App
	)

	it.After(func() {
		app = DestroyApp(app)
	})

	it("supplies apt packages to later buildpacks", func() {
		app = cutlass.New(settings.FixturePath)
		app.Buildpacks = []string{"apt_buildpack", "https://github.com/cloudfoundry/binary-buildpack#master"}
		app.SetEnv("BP_DEBUG", "1")

		PushAppAndConfirm(t, app)
		Expect(app.Stdout.String()).To(ContainSubstring("Installing apt packages"))

		// authenticating the apt packages
		Expect(app.Stdout.String()).NotTo(ContainSubstring("The following packages cannot be authenticated"))

		// installing packages from the default repo
		Expect(app.GetBody("/bosh")).To(ContainSubstring("BOSH: version 2"))

		// installing packages from a specific file location
		Expect(app.GetBody("/jq")).To(ContainSubstring("Jq: jq-1."))

		// installing a package from a specific repository with a lower priority
		Expect(app.GetBody("/cf")).To(ContainSubstring("cf version 6.38.0+7ddf0aadd.2018-08-07"))
	})
}
