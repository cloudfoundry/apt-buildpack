package integration_test

import (
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Apt supply buildpack", func() {
	var (
		app         *cutlass.App
		repo        *cutlass.App
		appDir      string
		repoBaseUrl string
		err         error
		cleanASG    func() ([]byte, error)
	)

	templateFile := func(templatePath, outputPath string, values map[string]string) *template.Template {
		template, err := template.ParseFiles(templatePath)
		Expect(err).ToNot(HaveOccurred())
		file, err := os.Create(outputPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(template.Execute(file, values)).To(Succeed())
		Expect(file.Close()).To(Succeed())
		return nil
	}

	BeforeEach(func() {
		repo = cutlass.New(filepath.Join(bpDir, "fixtures", "repo"))
		repo.Buildpacks = []string{"https://github.com/cloudfoundry/staticfile-buildpack#master"}
		Expect(repo.Push()).To(Succeed())
		Eventually(func() ([]string, error) { return repo.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

		appDir, err = cutlass.CopyFixture(filepath.Join(bpDir, "fixtures", "simple"))
		Expect(err).NotTo(HaveOccurred())

		repoBaseUrl, err = repo.GetUrl("/")
		Expect(err).NotTo(HaveOccurred())

		templateFile(filepath.Join(bpDir, "fixtures", "simple", "apt.yml"),
			filepath.Join(appDir, "apt.yml"),
			map[string]string{"repoBaseURL": repoBaseUrl})
	})

	AfterEach(func() {
		app = DestroyApp(app)
		repo = DestroyApp(repo)

		Expect(os.RemoveAll(appDir)).To(Succeed())
	})

	Context("as a supply buildpack", func() {
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
			cutlass.ApiVersion()
		})
	})

	Context("when using a private apt repo", func() {
		BeforeEach(func() {
			cleanASG, err = SetStagingASG(filepath.Join(bpDir, "fixtures", "asg_config.json"))
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			b, err := cleanASG()
			Expect(err).NotTo(HaveOccurred(), string(b))
		})

		It("doesn't navigate to canonical", func() {
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

func SetStagingASG(ASGConfigPath string) (func() ([]byte, error), error) {
	setASG := fmt.Sprintf(`cf create-security-group test_asg %s && cf bind-staging-security-group test_asg && cf unbind-staging-security-group public_networks`, ASGConfigPath)

	b, err := exec.Command("bash", "-c", setASG).CombinedOutput()
	if err != nil {
		return func() ([]byte, error) { return b, nil }, errors.Wrap(err, string(b))
	}

	clearASG := `cf bind-staging-security-group public_networks && cf unbind-staging-security-group test_asg`

	return func() ([]byte, error) { return exec.Command("bash", "-c", clearASG).CombinedOutput() }, nil
}
