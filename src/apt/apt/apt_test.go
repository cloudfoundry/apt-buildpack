package apt_test

import (
	"apt/apt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=apt.go --destination=mocks_test.go --package=apt_test

var _ = Describe("Apt", func() {
	var (
		a           *apt.Apt
		aptfile     *os.File
		mockCtrl    *gomock.Controller
		mockCommand *MockCommand
		cacheDir    string
		installDir  string
	)
	BeforeEach(func() {
		var err error
		aptfile, err = ioutil.TempFile("", "aptfile")
		Expect(err).ToNot(HaveOccurred())

		cacheDir, _ = ioutil.TempDir("", "cachedir")
		installDir, _ = ioutil.TempDir("", "installdir")

		mockCtrl = gomock.NewController(GinkgoT())
		mockCommand = NewMockCommand(mockCtrl)
	})
	JustBeforeEach(func() {
		var err error
		a, err = apt.New(mockCommand, aptfile.Name(), cacheDir, installDir)
		Expect(err).ToNot(HaveOccurred())
	})
	AfterEach(func() {
		aptfile.Close()
		os.Remove(aptfile.Name())
		os.RemoveAll(cacheDir)
		mockCtrl.Finish()
	})

	Describe("Update", func() {
		It("runs apt update with expected options", func() {
			mockCommand.EXPECT().Output(
				"/", "apt-get",
				"-o", "debug::nolocking=true",
				"-o", "dir::cache="+cacheDir+"/apt/cache",
				"-o", "dir::state="+cacheDir+"/apt/state",
				"-o", "dir::etc::sourcelist="+cacheDir+"/apt/sources/sources.list",
				"-o", "dir::etc::trusted="+cacheDir+"/apt/etc/trusted.gpg",
				"update",
			).Return("Shell output", nil)

			output, err := a.Update()
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal("Shell output"))
		})
	})

	Describe("Download", func() {
		BeforeEach(func() {
			aptfile.WriteString("http://example.com/holiday.deb\ndisneyland\n")
			aptfile.Close()
		})
		It("downloads user specified packages", func() {
			packageFile := cacheDir + "/apt/cache/archives/holiday.deb"
			mockCommand.EXPECT().Output(
				"/", "curl", "-s", "-L",
				"-z", packageFile,
				"-o", packageFile,
				"http://example.com/holiday.deb",
			).Return("curl output", nil)
			mockCommand.EXPECT().Output(
				"/", "apt-get",
				"-o", "debug::nolocking=true",
				"-o", "dir::cache="+cacheDir+"/apt/cache",
				"-o", "dir::state="+cacheDir+"/apt/state",
				"-o", "dir::etc::sourcelist="+cacheDir+"/apt/sources/sources.list",
				"-o", "dir::etc::trusted="+cacheDir+"/apt/etc/trusted.gpg",
				"-y", "--force-yes", "-d", "install", "--reinstall",
				"disneyland",
			).Return("apt output", nil)
			Expect(a.Download()).To(Equal(""))
		})
	})

	Describe("Install", func() {
		BeforeEach(func() {
			var err error
			cacheDir, err = ioutil.TempDir("", "cachedir")
			Expect(err).ToNot(HaveOccurred())
			Expect(os.MkdirAll(filepath.Join(cacheDir, "apt", "cache", "archives"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(cacheDir, "apt", "cache", "archives", "holiday.deb"), []byte{}, 0644)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(cacheDir, "apt", "cache", "archives", "disneyland.deb"), []byte{}, 0644)).To(Succeed())
		})
		It("installs the downloaded debs", func() {
			mockCommand.EXPECT().Output("/", "dpkg", "-x", filepath.Join(cacheDir, "apt", "cache", "archives", "holiday.deb"), installDir)
			mockCommand.EXPECT().Output("/", "dpkg", "-x", filepath.Join(cacheDir, "apt", "cache", "archives", "disneyland.deb"), installDir)
			Expect(a.Install()).To(Equal(""))
		})
	})
})
