package integration_test

import (
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPrivateRepo(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		app      *cutlass.App
		repo     *cutlass.App
		appDir   string
		cleanASG func() ([]byte, error)
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

		cleanASG, err = SetStagingASG(filepath.Join(bpDir, "fixtures", "asg_config.json"))
		Expect(err).ToNot(HaveOccurred())
	})

	it.After(func() {
		b, err := cleanASG()
		Expect(err).NotTo(HaveOccurred(), string(b))

		app = DestroyApp(app)
		repo = DestroyApp(repo)

		Expect(os.RemoveAll(appDir)).To(Succeed())
	})

	it("doesn't navigate to canonical", func() {
		app = cutlass.New(appDir)
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

func SetStagingASG(ASGConfigPath string) (func() ([]byte, error), error) {
	setASG := fmt.Sprintf(`cf create-security-group test_asg %s && cf bind-staging-security-group test_asg && cf unbind-staging-security-group public_networks`, ASGConfigPath)

	b, err := exec.Command("bash", "-c", setASG).CombinedOutput()
	if err != nil {
		return func() ([]byte, error) { return b, nil }, errors.Wrap(err, string(b))
	}

	clearASG := `cf bind-staging-security-group public_networks && cf unbind-staging-security-group test_asg`

	return func() ([]byte, error) { return exec.Command("bash", "-c", clearASG).CombinedOutput() }, nil
}
