package integration_test

import (
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Apt supply buildpack", func() {
	var app *cutlass.App
	AfterEach(func() { app = DestroyApp(app) })

	Context("as a supply buildpack", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "simple"))
			app.Buildpacks = []string{"apt_buildpack", "binary_buildpack"}
			app.SetEnv("BP_DEBUG", "1")
		})

		It("supplies apt packages to later buildpacks", func() {
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(ContainSubstring("Installing apt packages"))
			Expect(app.GetBody("/")).To(ContainSubstring("Ascii: ASCII 6/4 is decimal 100, hex 64"))
			Expect(app.GetBody("/jq")).To(ContainSubstring("Jq: jq-1."))
			Expect(app.GetBody("/bosh")).To(ContainSubstring("BOSH: version 2"))
		})
	})

	Context("as a final buildpack", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "simple"))
			app.Buildpacks = []string{"binary_buildpack", "apt_buildpack"}
			app.SetEnv("BP_DEBUG", "1")
		})

		It("reports failure", func() {
			Expect(app.Push()).To(HaveOccurred())
			Eventually(app.Stdout.String, 3*time.Second).Should(MatchRegexp("(?i)failed"))

			Expect(app.Stdout.String()).To(ContainSubstring("Warning: the last buildpack is not compatible with multi-buildpack apps"))
		})
	})

	Context("as a single buildpack", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "simple"))
			app.Buildpacks = []string{"apt_buildpack"}
			app.SetEnv("BP_DEBUG", "1")
		})

		It("reports failure", func() {
			Expect(app.Push()).To(HaveOccurred())
			Eventually(app.Stdout.String, 3*time.Second).Should(MatchRegexp("(?i)failed"))

			Expect(app.Stdout.String()).To(ContainSubstring("Warning: this buildpack can only be run as a supply buildpack, it can not be run alone"))
		})
	})
})
