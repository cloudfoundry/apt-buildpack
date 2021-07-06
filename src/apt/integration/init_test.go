package integration_test

import (
	"flag"
	"html/template"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var settings struct {
	FixturePath      string
	BuildpackPath    string
	BuildpackVersion string
}

func init() {
	flag.StringVar(&settings.BuildpackVersion, "version", "", "version to use (builds if empty)")
	flag.BoolVar(&cutlass.Cached, "cached", true, "cached buildpack")
	flag.StringVar(&cutlass.DefaultMemory, "memory", "128M", "default memory for pushed apps")
	flag.StringVar(&cutlass.DefaultDisk, "disk", "256M", "default disk for pushed apps")
}

func TestIntegration(t *testing.T) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		packagedBuildpack cutlass.VersionedBuildpackPackage
		err               error
	)

	if settings.BuildpackVersion == "" {
		packagedBuildpack, err = cutlass.PackageUniquelyVersionedBuildpack(os.Getenv("CF_STACK"), true)
		Expect(err).NotTo(HaveOccurred())

		settings.BuildpackVersion = packagedBuildpack.Version
	}

	settings.BuildpackPath, err = cutlass.FindRoot()
	Expect(err).NotTo(HaveOccurred())

	Expect(cutlass.CopyCfHome()).To(Succeed())
	cutlass.SeedRandom()

	repo := cutlass.New(filepath.Join(settings.BuildpackPath, "fixtures", "repo"))
	defer repo.Destroy()

	repo.Buildpacks = []string{"https://github.com/cloudfoundry/staticfile-buildpack#master"}
	Expect(repo.Push()).To(Succeed())
	Eventually(func() ([]string, error) { return repo.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

	url, err := repo.GetUrl("/")
	Expect(err).NotTo(HaveOccurred())

	template, err := template.ParseFiles(filepath.Join(settings.BuildpackPath, "fixtures", "simple", "apt.yml"))
	Expect(err).ToNot(HaveOccurred())

	settings.FixturePath, err = cutlass.CopyFixture(filepath.Join(settings.BuildpackPath, "fixtures", "simple"))
	Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(settings.FixturePath)

	file, err := os.Create(filepath.Join(settings.FixturePath, "apt.yml"))
	Expect(err).NotTo(HaveOccurred())

	Expect(template.Execute(file, map[string]string{"repoBaseURL": url})).To(Succeed())
	Expect(file.Close()).To(Succeed())

	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Default", testDefault)
	suite("PrivateRepo", testPrivateRepo)
	suite("Failure", testFailure)
	suite.Run(t)

	Expect(cutlass.RemovePackagedBuildpack(packagedBuildpack)).To(Succeed())
	Expect(cutlass.DeleteOrphanedRoutes()).To(Succeed())
}

func PushAppAndConfirm(t *testing.T, app *cutlass.App) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually
	)

	Expect(app.Push()).To(Succeed())
	Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
	Expect(app.ConfirmBuildpack(settings.BuildpackVersion)).To(Succeed())
}

func DestroyApp(app *cutlass.App) *cutlass.App {
	if app != nil {
		app.Destroy()
	}

	return nil
}
