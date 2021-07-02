package integration_test

import (
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("when used inappropriately", func() {
	var (
		app         *cutlass.App
		repo        *cutlass.App
		appDir      string
		repoBaseURL string
	)

	BeforeEach(func() {
		repo = cutlass.New(filepath.Join(bpDir, "fixtures", "repo"))
		repo.Buildpacks = []string{"https://github.com/cloudfoundry/staticfile-buildpack#master"}
		Expect(repo.Push()).To(Succeed())
		Eventually(func() ([]string, error) { return repo.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

		var err error
		appDir, err = cutlass.CopyFixture(filepath.Join(bpDir, "fixtures", "simple"))
		Expect(err).NotTo(HaveOccurred())

		repoBaseURL, err = repo.GetUrl("/")
		Expect(err).NotTo(HaveOccurred())

		templatePath := filepath.Join(bpDir, "fixtures", "simple", "apt.yml")
		outputPath := filepath.Join(appDir, "apt.yml")
		values := map[string]string{"repoBaseURL": repoBaseURL}

		template, err := template.ParseFiles(templatePath)
		Expect(err).ToNot(HaveOccurred())

		file, err := os.Create(outputPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(template.Execute(file, values)).To(Succeed())
		Expect(file.Close()).To(Succeed())
	})

	AfterEach(func() {
		app = DestroyApp(app)
		repo = DestroyApp(repo)

		Expect(os.RemoveAll(appDir)).To(Succeed())
	})

	Context("as a final buildpack", func() {
		BeforeEach(func() {
			app = cutlass.New(appDir)
			app.Buildpacks = []string{"https://github.com/cloudfoundry/binary-buildpack#master", "apt_buildpack"}
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
			app = cutlass.New(appDir)
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
