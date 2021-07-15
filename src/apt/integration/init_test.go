package integration_test

import (
	"flag"
	"html/template"
	"os"
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
}

func init() {
	flag.BoolVar(&settings.Cached, "cached", false, "run cached buildpack tests")
	flag.BoolVar(&settings.Serial, "serial", false, "run serial buildpack tests")
	flag.StringVar(&settings.GitHubToken, "github-token", "", "use the token to make GitHub API requests")
	flag.StringVar(&settings.Platform, "platform", "cf", `switchblade platform to test against ("cf" or "docker")`)
}

func TestIntegration(t *testing.T) {
	Expect := NewWithT(t).Expect
	format.MaxLength = 0
	SetDefaultEventuallyTimeout(10 * time.Second)

	platform, err := switchblade.NewPlatform(settings.Platform, settings.GitHubToken)
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
}
