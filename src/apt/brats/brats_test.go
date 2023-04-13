package brats_test

import (
	"html/template"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	"github.com/cloudfoundry/libbuildpack/bratshelper"
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

	AfterEach(func() {
		app = DestroyApp(app)
		repo = DestroyApp(repo)
		Expect(os.RemoveAll(appDir)).To(Succeed())
	})

	Context("Unbuilt buildpack (eg github)", func() {
		BeforeEach(func() {
			repo = cutlass.New(filepath.Join(bratshelper.Data.BpDir, "fixtures", "repo"))
			repo.Buildpacks = []string{"https://github.com/cloudfoundry/staticfile-buildpack#master"}
			PushApp(repo)

			appDir, err = cutlass.CopyFixture(filepath.Join(bratshelper.Data.BpDir, "fixtures", "simple"))
			Expect(err).NotTo(HaveOccurred())

			repoBaseUrl, err := repo.GetUrl("/")
			Expect(err).NotTo(HaveOccurred())

			aptYamlTemplate := template.New("apt.yml")
			_, err = aptYamlTemplate.ParseFiles(filepath.Join(bratshelper.Data.BpDir, "fixtures", "simple", "apt.yml"))
			Expect(err).ToNot(HaveOccurred())

			aptYaml, err := os.Create(filepath.Join(appDir, "apt.yml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(aptYamlTemplate.Execute(aptYaml, map[string]string{"repoBaseURL": repoBaseUrl})).To(Succeed())
			Expect(aptYaml.Close()).To(Succeed())

			app = cutlass.New(appDir)
			app.Buildpacks = []string{bratshelper.Data.Uncached, rubyBuildpackName, "https://github.com/cloudfoundry/binary-buildpack#master"}
		})

		It("runs", func() {
			PushApp(app)
			Expect(app.Stdout.String()).To(ContainSubstring("-----> Download go"))

			Expect(app.Stdout.String()).To(ContainSubstring("Installing apt packages"))
			Expect(app.GetBody("/bosh")).To(ContainSubstring("BOSH: version 2"))
		})
	})
})
