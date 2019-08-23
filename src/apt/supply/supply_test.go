package supply_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/apt-buildpack/src/apt/supply"

	"bytes"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=supply.go --destination=mocks_test.go --package=supply_test

var _ = Describe("Supply", func() {
	var (
		depDir     string
		supplier   *supply.Supplier
		logger     *libbuildpack.Logger
		mockCtrl   *gomock.Controller
		mockStager *MockStager
		mockApt    *MockApt
		buffer     *bytes.Buffer
	)

	BeforeEach(func() {
		var err error
		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(buffer)

		mockCtrl = gomock.NewController(GinkgoT())
		mockStager = NewMockStager(mockCtrl)
		depDir, err = ioutil.TempDir("", "apt.depdir")
		Expect(err).ToNot(HaveOccurred())
		mockStager.EXPECT().DepDir().AnyTimes().Return(depDir)
		mockApt = NewMockApt(mockCtrl)
	})

	JustBeforeEach(func() {
		supplier = supply.New(mockStager, mockApt, logger)
	})

	AfterEach(func() {
		mockCtrl.Finish()
		os.RemoveAll(depDir)
	})

	allowAllAptMethods := func() {
		mockApt.EXPECT().Setup().AnyTimes()
		mockApt.EXPECT().HasKeys().AnyTimes()
		mockApt.EXPECT().HasRepos().AnyTimes()
		mockApt.EXPECT().HasClean().AnyTimes()
		mockApt.EXPECT().Update().AnyTimes()
		mockApt.EXPECT().DownloadAll().AnyTimes()
		mockApt.EXPECT().InstallAll().AnyTimes()
	}

	allowAllDepLinkingMethods := func() {
		mockStager.EXPECT().LinkDirectoryInDepDir(gomock.Any(), gomock.Any()).AnyTimes()
	}

	Describe("Run", func() {
		It("install the apt packages", func() {
			gomock.InOrder(
				mockApt.EXPECT().Setup(),
				mockApt.EXPECT().HasKeys(),
				mockApt.EXPECT().HasRepos(),
				mockApt.EXPECT().HasClean(),
				mockApt.EXPECT().Update(),
				mockApt.EXPECT().DownloadAll(),
				mockApt.EXPECT().InstallAll(),
			)
			allowAllDepLinkingMethods()
			Expect(supplier.Run()).To(Succeed())
		})

		It("symlinks the apt packages", func() {
			allowAllAptMethods()
			Expect(os.MkdirAll(filepath.Join(depDir, "apt", "usr", "bin"), 0755)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(depDir, "apt", "lib", "x86_64-linux-gnu"), 0755)).To(Succeed())

			mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(depDir, "apt", "usr", "bin"), "bin")
			mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(depDir, "apt", "lib", "x86_64-linux-gnu"), "lib")

			Expect(supplier.Run()).To(Succeed())
		})

		Context("Aptfile has keys", func() {
			It("Calls AddKeys", func() {
				gomock.InOrder(
					mockApt.EXPECT().Setup(),
					mockApt.EXPECT().HasKeys().Return(true),
					mockApt.EXPECT().AddKeys(),
					mockApt.EXPECT().HasRepos(),
					mockApt.EXPECT().HasClean(),
					mockApt.EXPECT().Update(),
					mockApt.EXPECT().DownloadAll(),
					mockApt.EXPECT().InstallAll(),
				)
				allowAllDepLinkingMethods()
				Expect(supplier.Run()).To(Succeed())
			})
		})

		Context("Aptfile has repos", func() {
			It("Calls AddRepos", func() {
				gomock.InOrder(
					mockApt.EXPECT().Setup(),
					mockApt.EXPECT().HasKeys(),
					mockApt.EXPECT().HasRepos().Return(true),
					mockApt.EXPECT().AddRepos(),
					mockApt.EXPECT().HasClean(),
					mockApt.EXPECT().Update(),
					mockApt.EXPECT().DownloadAll(),
					mockApt.EXPECT().InstallAll(),
				)
				allowAllDepLinkingMethods()
				Expect(supplier.Run()).To(Succeed())
			})
		})
	})
})
