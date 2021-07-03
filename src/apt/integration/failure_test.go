package integration_test

import (
	"html/template"
	"os"
	"path/filepath"
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

		app    *cutlass.App
		repo   *cutlass.App
		appDir string
	)

	it.Before(func() {
		repo = cutlass.New(filepath.Join(bpDir, "fixtures", "repo"))
		repo.Buildpacks = []string{"https://github.com/cloudfoundry/staticfile-buildpack#master"}
		Expect(repo.Push()).To(Succeed())
		Eventually(func() ([]string, error) { return repo.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

		var err error
		appDir, err = cutlass.CopyFixture(filepath.Join(bpDir, "fixtures", "simple"))
		Expect(err).NotTo(HaveOccurred())

		repoBaseURL, err := repo.GetUrl("/")
		Expect(err).NotTo(HaveOccurred())

		template, err := template.ParseFiles(filepath.Join(bpDir, "fixtures", "simple", "apt.yml"))
		Expect(err).ToNot(HaveOccurred())

		file, err := os.Create(filepath.Join(appDir, "apt.yml"))
		Expect(err).NotTo(HaveOccurred())

		Expect(template.Execute(file, map[string]string{"repoBaseURL": repoBaseURL})).To(Succeed())
		Expect(file.Close()).To(Succeed())
	})

	it.After(func() {
		app = DestroyApp(app)
		repo = DestroyApp(repo)

		Expect(os.RemoveAll(appDir)).To(Succeed())
	})

	context("as a final buildpack", func() {
		it.Before(func() {
			app = cutlass.New(appDir)
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
			app = cutlass.New(appDir)
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
