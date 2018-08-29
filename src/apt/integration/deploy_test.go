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

var _ = Describe("Apt supply buildpack", func() {
	var (
		app    *cutlass.App
		repo   *cutlass.App
		appDir string
		err    error
	)

	BeforeEach(func() {
		repo = cutlass.New(filepath.Join(bpDir, "fixtures", "repo"))
		repo.Buildpacks = []string{"https://github.com/cloudfoundry/staticfile-buildpack#master"}
		Expect(repo.Push()).To(Succeed())
		Eventually(func() ([]string, error) { return repo.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

		appDir, err = cutlass.CopyFixture(filepath.Join(bpDir, "fixtures", "simple"))
		Expect(err).NotTo(HaveOccurred())

		repoBaseUrl, err := repo.GetUrl("/")
		Expect(err).NotTo(HaveOccurred())

		aptYamlTemplate := template.New("apt.yml")
		_, err = aptYamlTemplate.ParseFiles(filepath.Join(bpDir, "fixtures", "simple", "apt.yml"))
		Expect(err).ToNot(HaveOccurred())

		aptYaml, err := os.Create(filepath.Join(appDir, "apt.yml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(aptYamlTemplate.Execute(aptYaml, map[string]string{"repoBaseURL": repoBaseUrl})).To(Succeed())
		Expect(aptYaml.Close()).To(Succeed())
	})

	AfterEach(func() {
		app = DestroyApp(app)
		repo = DestroyApp(repo)

		Expect(os.RemoveAll(appDir)).To(Succeed())
	})

	Context("as a supply buildpack", func() {
		BeforeEach(func() {
			app = cutlass.New(appDir)
			app.Buildpacks = []string{"apt_buildpack", "https://github.com/cloudfoundry/binary-buildpack#master"}
			app.SetEnv("BP_DEBUG", "1")
		})

		It("supplies apt packages to later buildpacks", func() {
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(ContainSubstring("Installing apt packages"))
			Expect(app.GetBody("/bosh")).To(ContainSubstring("BOSH: version 2"))
			Expect(app.GetBody("/jq")).To(ContainSubstring("Jq: jq-1."))
			Expect(app.GetBody("/zsh")).To(ContainSubstring("zsh 5.0.5"))
		})
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
