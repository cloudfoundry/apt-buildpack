package integration_test

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/switchblade"
	"github.com/onsi/gomega/format"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var settings struct {
	Cached      bool
	Serial      bool
	GitHubToken string
	Platform    string
	Stack       string
}

func init() {
	flag.BoolVar(&settings.Cached, "cached", false, "run cached buildpack tests")
	flag.BoolVar(&settings.Serial, "serial", false, "run serial buildpack tests")
	flag.StringVar(&settings.GitHubToken, "github-token", "", "use the token to make GitHub API requests")
	flag.StringVar(&settings.Platform, "platform", "cf", `switchblade platform to test against ("cf" or "docker")`)
	flag.StringVar(&settings.Stack, "stack", "cflinuxfs3", "stack to use when pushing apps")
}

func TestIntegration(t *testing.T) {
	Expect := NewWithT(t).Expect
	format.MaxLength = 0
	SetDefaultEventuallyTimeout(10 * time.Second)

	platform, err := switchblade.NewPlatform(settings.Platform, settings.GitHubToken, settings.Stack)
	Expect(err).NotTo(HaveOccurred())

	root, err := filepath.Abs("./../../..")
	Expect(err).NotTo(HaveOccurred())

	err = platform.Initialize(switchblade.Buildpack{
		Name: "apt_buildpack",
		URI:  os.Getenv("BUILDPACK_FILE"),
	})
	Expect(err).NotTo(HaveOccurred())

	repoName, err := switchblade.RandomName()
	Expect(err).NotTo(HaveOccurred())

	// tech debt alert!
	// remove this block when testing envs come with cflinuxfs4
	// enabled staticfile & binary buildpacks.
	if settings.Stack == "cflinuxfs4" {
		err = platform.Initialize(switchblade.Buildpack{
			Name: "staticfile_buildpack",
			URI:  "https://github.com/cloudfoundry/staticfile-buildpack/releases/download/v1.6.0/staticfile-buildpack-cflinuxfs4-v1.6.0.zip",
		})
		Expect(err).NotTo(HaveOccurred())

		err = platform.Initialize(switchblade.Buildpack{
			Name: "binary_buildpack",
			URI:  "https://github.com/cloudfoundry/binary-buildpack/releases/download/v1.1.3/binary-buildpack-cflinuxfs4-v1.1.3.zip",
		})
		Expect(err).NotTo(HaveOccurred())
	}

	rubyTmpDir, err := os.MkdirTemp("", "ruby")
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create tempdir: %v", err))

	// We need a cached ruby-buildpack to run the simple web app in offline mode
	// This builds a cached ruby-builpack to ${tmpDir}/ruby-buidpack.zip
	command := exec.Command("scripts/build-ruby-offline-bp.sh", "--stack", settings.Stack, "--outputDir", rubyTmpDir)
	command.Dir = root
	data, err := command.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create cached ruby_buildpack:\n%s\n%v", string(data), err))

	err = platform.Initialize(switchblade.Buildpack{
		Name: "ruby_buildpack",
		URI:  filepath.Join(rubyTmpDir, "ruby-buildpack.zip"),
	})
	Expect(err).NotTo(HaveOccurred())

	repoDeployment, _, err := platform.Deploy.
		WithBuildpacks("staticfile_buildpack").
		Execute(repoName, filepath.Join(root, "fixtures", "repo"))
	Expect(err).NotTo(HaveOccurred())

	template, err := template.ParseFiles(filepath.Join(root, "fixtures", "simple", "apt.yml"))
	Expect(err).NotTo(HaveOccurred())

	fixturePath, err := switchblade.Source(filepath.Join(root, "fixtures", "simple"))
	Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(fixturePath)

	file, err := os.Create(filepath.Join(fixturePath, "apt.yml"))
	Expect(err).NotTo(HaveOccurred())

	Expect(template.Execute(file, map[string]string{"repoBaseURL": repoDeployment.InternalURL})).To(Succeed())
	Expect(file.Close()).To(Succeed())

	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Default", testDefault(platform, fixturePath))
	suite("PrivateRepo", testPrivateRepo(platform, fixturePath))
	suite("Failure", testFailure(platform, fixturePath))
	suite.Run(t)

	Expect(platform.Delete.Execute(repoName)).To(Succeed())
	Expect(os.Remove(os.Getenv("BUILDPACK_FILE"))).To(Succeed())
	Expect(os.RemoveAll(rubyTmpDir)).To(Succeed())
}
