package brats_test

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"strings"

	"github.com/cloudfoundry/libbuildpack/bratshelper"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var rubyTmpDir, rubyBuildpackName string

var _ = func() bool {
	testing.Init()
	return true
}()

func init() {
	flag.StringVar(&cutlass.DefaultMemory, "memory", "128M", "default memory for pushed apps")
	flag.StringVar(&cutlass.DefaultDisk, "disk", "256M", "default disk for pushed apps")
	flag.Parse()
}

var _ = SynchronizedBeforeSuite(func() []byte {
	// Run once
	var err error
	rubyTmpDir, err = os.MkdirTemp("", "ruby")
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create tempdir: %v", err))
	root, err := filepath.Abs("./../../..")
	Expect(err).NotTo(HaveOccurred())

	// We need a cached ruby-buildpack to run the simple web app in offline mode
	// This builds a cached ruby-builpack to ${tmpDir}/ruby-buidpack.zip
	command := exec.Command("scripts/build-ruby-offline-bp.sh", "--stack", os.Getenv("CF_STACK"), "--outputDir", rubyTmpDir)
	command.Dir = root
	data, err := command.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create cached ruby_buildpack:\n%s\n%v", string(data), err))
	rubyBuildpackName = fmt.Sprintf("%s_buildpack", filepath.Base(rubyTmpDir))
	command = exec.Command("cf", "create-buildpack", rubyBuildpackName, filepath.Join(rubyTmpDir, "ruby-buildpack.zip"), "1", "--enable")
	data, err = command.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create %s_buildpack:\n%s\n%v", rubyTmpDir, string(data), err))
	return bratshelper.InitBpData(os.Getenv("CF_STACK"), true).Marshal()
}, func(data []byte) {
	// Run on all nodes
	bratshelper.Data.Unmarshal(data)
	Expect(cutlass.CopyCfHome()).To(Succeed())
	cutlass.SeedRandom()
	cutlass.DefaultStdoutStderr = GinkgoWriter
})

var _ = SynchronizedAfterSuite(func() {
	// Run on all nodes
}, func() {
	// Run once
	Expect(cutlass.DeleteOrphanedRoutes()).To(Succeed())
	Expect(cutlass.DeleteBuildpack(strings.Replace(bratshelper.Data.Cached, "_buildpack", "", 1))).To(Succeed())
	Expect(cutlass.DeleteBuildpack(strings.Replace(bratshelper.Data.Uncached, "_buildpack", "", 1))).To(Succeed())
	Expect(cutlass.DeleteBuildpack(rubyBuildpackName)).To(Succeed())
	Expect(os.Remove(bratshelper.Data.CachedFile)).To(Succeed())
	Expect(os.Remove(bratshelper.Data.UncachedFile)).To(Succeed())
	Expect(os.RemoveAll(rubyTmpDir)).To(Succeed())
})

func TestBrats(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Brats Suite")
}

func PushApp(app *cutlass.App) {
	Expect(app.Push()).To(Succeed())
	Eventually(app.InstanceStates, 20*time.Second).Should(Equal([]string{"RUNNING"}))
}

func DestroyApp(app *cutlass.App) *cutlass.App {
	if app != nil {
		app.Destroy()
	}
	return nil
}
