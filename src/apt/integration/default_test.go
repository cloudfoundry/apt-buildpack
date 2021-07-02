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

var _ = Describe("as a supply buildpack", func() {
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

	It("supplies apt packages to later buildpacks", func() {
		app = cutlass.New(appDir)
		app.Buildpacks = []string{"apt_buildpack", "https://github.com/cloudfoundry/binary-buildpack#master"}
		app.SetEnv("BP_DEBUG", "1")

		PushAppAndConfirm(app)
		Expect(app.Stdout.String()).To(ContainSubstring("Installing apt packages"))

		By("authenticating the apt packages")
		Expect(app.Stdout.String()).NotTo(ContainSubstring("The following packages cannot be authenticated"))

		By("installing packages from the default repo")
		Expect(app.GetBody("/bosh")).To(ContainSubstring("BOSH: version 2"))

		By("installing packages from a specific file location")
		Expect(app.GetBody("/jq")).To(ContainSubstring("Jq: jq-1."))

		By("installing a package from a specific repository with a lower priority")
		Expect(app.GetBody("/cf")).To(ContainSubstring("cf version 6.38.0+7ddf0aadd.2018-08-07"))
	})
})
