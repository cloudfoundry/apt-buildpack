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

var rubyBuildpackFolder, staticfileBuildpackFolder, binaryBuildpackFolder string

func init() {
	flag.BoolVar(&settings.Cached, "cached", false, "run cached buildpack tests")
	flag.BoolVar(&settings.Serial, "serial", false, "run serial buildpack tests")
	flag.StringVar(&settings.GitHubToken, "github-token", "", "use the token to make GitHub API requests")
	flag.StringVar(&settings.Platform, "platform", "cf", `switchblade platform to test against ("cf" or "docker")`)
	flag.StringVar(&settings.Stack, "stack", "cflinuxfs4", "stack to use when pushing apps")
}

func TestIntegration(t *testing.T) {
	Expect := NewWithT(t).Expect
	format.MaxLength = 0
	SetDefaultEventuallyTimeout(10 * time.Second)

	root, err := filepath.Abs("./../../..")
	Expect(err).NotTo(HaveOccurred())

	platform, err := switchblade.NewPlatform(settings.Platform, settings.GitHubToken, settings.Stack)
	Expect(err).NotTo(HaveOccurred())

	rubyBuildpackFolder, err = prepareRequiredBuildpack("ruby", root)
	Expect(err).NotTo(HaveOccurred())

	err = platform.Initialize(
		switchblade.Buildpack{
			Name: "apt_buildpack",
			URI:  os.Getenv("BUILDPACK_FILE"),
		},
		switchblade.Buildpack{
			Name: "ruby_buildpack",
			URI:  filepath.Join(rubyBuildpackFolder, "ruby-buildpack.zip"),
		},
	)
	Expect(err).NotTo(HaveOccurred())

	if os.Getenv("CF_STACK") == "cflinuxfs3" || settings.Platform == "docker" {
		staticfileBuildpackFolder, err = prepareRequiredBuildpack("staticfile", root)
		Expect(err).NotTo(HaveOccurred())

		binaryBuildpackFolder, err = prepareRequiredBuildpack("binary", root)
		Expect(err).NotTo(HaveOccurred())

		err = platform.Initialize(
			switchblade.Buildpack{
				Name: "staticfile_buildpack",
				URI:  filepath.Join(staticfileBuildpackFolder, "staticfile-buildpack.zip"),
			},
			switchblade.Buildpack{
				Name: "binary_buildpack",
				URI:  filepath.Join(binaryBuildpackFolder, "binary-buildpack.zip"),
			},
		)
		Expect(err).NotTo(HaveOccurred())
	}

	repoName, err := switchblade.RandomName()
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
	Expect(os.RemoveAll(rubyBuildpackFolder)).To(Succeed())
	Expect(platform.Deinitialize()).To(Succeed())

	if os.Getenv("CF_STACK") == "cflinuxfs3" {
		Expect(os.RemoveAll(staticfileBuildpackFolder)).To(Succeed())
		Expect(os.RemoveAll(binaryBuildpackFolder)).To(Succeed())
	}
}

func prepareRequiredBuildpack(buildpack, root string) (string, error) {
	buildpackTmpDir, err := os.MkdirTemp("", buildpack)
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %s", err)
	}

	command := exec.Command("scripts/build-offline-bp.sh", "--buildpack", buildpack, "--stack", settings.Stack, "--outputDir", buildpackTmpDir)
	command.Dir = root
	data, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to prepare buildpack: %s", data)
	}

	return buildpackTmpDir, nil
}
