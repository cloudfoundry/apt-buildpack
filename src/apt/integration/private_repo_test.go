package integration_test

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPrivateRepo(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		app      *cutlass.App
		cleanASG func() ([]byte, error)
	)

	it.Before(func() {
		var err error
		cleanASG, err = SetStagingASG(filepath.Join(settings.BuildpackPath, "fixtures", "asg_config.json"))
		Expect(err).ToNot(HaveOccurred())
	})

	it.After(func() {
		b, err := cleanASG()
		Expect(err).NotTo(HaveOccurred(), string(b))

		app = DestroyApp(app)
	})

	it("doesn't navigate to canonical", func() {
		app = cutlass.New(settings.FixturePath)
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
