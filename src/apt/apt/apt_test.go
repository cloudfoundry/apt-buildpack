package apt_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/apt-buildpack/src/apt/apt"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

//go:generate mockgen -source=apt.go --destination=mocks_test.go --package=apt_test

var _ = Describe("Apt", func() {
	var (
		a           *apt.Apt
		aptFile     string
		mockCtrl    *gomock.Controller
		mockCommand *MockCommand
		rootDir     string
		cacheDir    string
		installDir  string
	)

	BeforeEach(func() {
		bpDir, err := cutlass.FindRoot()
		Expect(err).NotTo(HaveOccurred())

		aptFile = filepath.Join(bpDir, "fixtures", "unit", "aptFile.yml")
		rootDir, _ = ioutil.TempDir("", "rootdir")
		cacheDir, _ = ioutil.TempDir("", "cachedir")
		installDir, _ = ioutil.TempDir("", "installdir")

		ioutil.WriteFile(filepath.Join(rootDir, "sources.list"), []byte(""), 0666)
		ioutil.WriteFile(filepath.Join(rootDir, "trusted.gpg"), []byte(""), 0666)
		ioutil.WriteFile(filepath.Join(rootDir, "preferences"), []byte(""), 0666)

		mockCtrl = gomock.NewController(GinkgoT())
		mockCommand = NewMockCommand(mockCtrl)
	})

	JustBeforeEach(func() {
		a = apt.New(mockCommand, aptFile, rootDir, cacheDir, installDir)
	})

	AfterEach(func() {
		os.Remove(aptFile)
		os.RemoveAll(cacheDir)
		os.RemoveAll(installDir)
		mockCtrl.Finish()
	})

	Describe("Setup", func() {
		JustBeforeEach(func() {
			content := &apt.Apt{
				GpgAdvancedOptions: []string{"--keyserver keys.gnupg.net --recv-keys 09617FD37CC06B54"},
				Keys:               []string{"https://example.com/public.key"},
				Repos: []apt.Repository{
					apt.Repository{Name: "deb http://apt.example.com stable main"},
					apt.Repository{Name: "foo bar baz", Priority: "100"},
				},
				Packages: []string{"abc", "def"},
			}
			Expect(libbuildpack.NewYAML().Write(aptFile, content)).To(Succeed())

			Expect(a.Setup()).To(Succeed())
		})

		It("sets keys from apt.yml", func() {
			Expect(a.Keys).To(Equal([]string{"https://example.com/public.key"}))
		})

		It("sets gpg advanced options from apt.yml", func() {
			Expect(a.GpgAdvancedOptions).To(Equal([]string{"--keyserver keys.gnupg.net --recv-keys 09617FD37CC06B54"}))
		})

		It("sets repos with priority from apt.yml", func() {
			Expect(a.Repos).To(Equal([]apt.Repository{
				apt.Repository{Name: "deb http://apt.example.com stable main"},
				apt.Repository{Name: "foo bar baz", Priority: "100"},
			}))
		})

		It("sets packages from apt.yml", func() {
			Expect(a.Packages).To(Equal([]string{"abc", "def"}))
		})

		It("copies sources.list", func() {
			Expect(filepath.Join(cacheDir, "apt", "sources", "sources.list")).To(BeARegularFile())
		})

		It("copies trusted.gpg", func() {
			copiedFile, err := libbuildpack.FileExists(filepath.Join(cacheDir, "apt", "etc", "trusted.gpg"))
			Expect(err).ToNot(HaveOccurred())
			copiedDir, err := libbuildpack.FileExists(filepath.Join(cacheDir, "apt", "etc", "trusted.gpg.d"))
			Expect(err).ToNot(HaveOccurred())
			Expect(copiedFile || copiedDir).To(BeTrue())
		})

		It("copies preferences", func() {
			Expect(filepath.Join(cacheDir, "apt", "etc", "preferences")).To(SatisfyAny(BeARegularFile(), Not(BeAnExistingFile())))
		})
	})

	Describe("HasKeys", func() {
		Context("GPG Advanced Options have been specified", func() {
			JustBeforeEach(func() {
				a.GpgAdvancedOptions = []string{"--keyserver keys.gnupg.net --recv-keys 09617FD37CC06B54"}
			})

			It("returns true from HasKeys()", func() {
				Expect(a.HasKeys()).To(BeTrue())
			})
		})

		Context("Keys have been specified", func() {
			JustBeforeEach(func() {
				a.Keys = []string{"https://example.com/public.key"}
			})

			It("returns true from HasKeys()", func() {
				Expect(a.HasKeys()).To(BeTrue())
			})
		})

		Context("Neither GPG Advanced Options nor Keys have been specfied", func() {
			It("returns false from HasKeys()", func() {
				Expect(a.HasKeys()).To(BeFalse())
			})
		})
	})

	Describe("AddKeys", func() {
		Context("GPG Advanced Options have been specified", func() {
			JustBeforeEach(func() {
				a.GpgAdvancedOptions = []string{"--keyserver keys.gnupg.net --recv-keys 09617FD37CC06B54"}
			})

			It("adds the keys to the apt trusted keys", func() {
				mockCommand.EXPECT().Output(
					"/", "apt-key",
					"--keyring", filepath.Join(cacheDir, "apt", "etc", "trusted.gpg"),
					"adv",
					"--keyserver keys.gnupg.net --recv-keys 09617FD37CC06B54",
				).Return("Shell output", nil)

				err := a.AddKeys()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("Keys have been specified", func() {
			JustBeforeEach(func() {
				a.Keys = []string{"https://example.com/public.key"}
			})

			It("adds the keys to the apt trusted keys", func() {
				mockCommand.EXPECT().Output(
					"/", "apt-key",
					"--keyring", filepath.Join(cacheDir, "apt", "etc", "trusted.gpg"),
					"adv",
					"--fetch-keys", "https://example.com/public.key",
				).Return("Shell output", nil)

				err := a.AddKeys()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("No keys specified", func() {
			JustBeforeEach(func() {
				a.Keys = []string{}
			})

			It("does nothing", func() {
				err := a.AddKeys()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("AddRepos", func() {
		Context("Keys and priorities have been specified", func() {
			var sourceList, prefFile string

			JustBeforeEach(func() {
				a.Repos = []apt.Repository{
					apt.Repository{Name: "repo 11"},
					apt.Repository{Name: "repo 12", Priority: "99"},
					apt.Repository{Name: "repo 13", Priority: "100"},
				}

				sourceList = filepath.Join(cacheDir, "apt", "sources", "sources.list")
				Expect(os.MkdirAll(filepath.Dir(sourceList), 0777)).To(Succeed())
				Expect(ioutil.WriteFile(sourceList, []byte("repo 1\nrepo 2"), 0666)).To(Succeed())

				prefFile = filepath.Join(cacheDir, "apt", "etc", "preferences")
				Expect(os.MkdirAll(filepath.Dir(prefFile), 0777)).To(Succeed())
				Expect(ioutil.WriteFile(prefFile, []byte("Package: *\nPin: release a=repo 1\nPin-Priority"), 0666)).To(Succeed())
			})

			It("adds the repos to the apt sources list", func() {
				Expect(a.AddRepos()).To(Succeed())

				content, err := ioutil.ReadFile(sourceList)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal("repo 1\nrepo 2\nrepo 11\nrepo 12\nrepo 13"))
			})

			It("adds repo priorities to the preferences file", func() {
				Expect(a.AddRepos()).To(Succeed())

				content, err := ioutil.ReadFile(prefFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal("Package: *\nPin: release a=repo 1\nPin-Priority\nPackage: *\nPin: release a=repo 12\nPin-Priority: 99\n\nPackage: *\nPin: release a=repo 13\nPin-Priority: 100\n"))
			})
		})

		Context("No keys specified", func() {
			JustBeforeEach(func() {
				a.Keys = []string{}
			})

			It("does nothing", func() {
				err := a.AddKeys()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("Update", func() {
		It("runs apt update with expected options", func() {
			mockCommand.EXPECT().Execute(
				"/", gomock.Any(), gomock.Any(), "apt-get",
				"-o", "debug::nolocking=true",
				"-o", "dir::cache="+cacheDir+"/apt/cache",
				"-o", "dir::state="+cacheDir+"/apt/state",
				"-o", "dir::etc::sourcelist="+cacheDir+"/apt/sources/sources.list",
				"-o", "dir::etc::trusted="+cacheDir+"/apt/etc/trusted.gpg",
				"-o", "Dir::Etc::preferences="+cacheDir+"/apt/etc/preferences",
				"update",
			).Return(nil)

			err := a.Update()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("DownloadAll", func() {
		var (
			fooFileName = "foo.deb"
			barFileName = "bar.deb"
			fooServer   *ghttp.Server
			barServer   *ghttp.Server
		)

		JustBeforeEach(func() {
			fooServer = ghttp.NewServer()
			fooServer.AppendHandlers(
				ghttp.VerifyRequest("GET", "/"+fooFileName),
			)
			fooFileUri := fooServer.URL() + "/" + fooFileName

			barServer = ghttp.NewServer()
			barServer.AppendHandlers(
				ghttp.VerifyRequest("GET", "/"+barFileName),
			)
			barFileUri := barServer.URL() + "/" + barFileName

			content := &apt.Apt{
				GpgAdvancedOptions: []string{"--keyserver keys.gnupg.net --recv-keys 09617FD37CC06B54"},
				Keys:               []string{"https://example.com/public.key"},
				Repos: []apt.Repository{
					apt.Repository{Name: "deb http://apt.example.com stable main"},
					apt.Repository{Name: "foo bar baz", Priority: "100"},
				},
				Packages: []string{"abc", "def"},
			}
			Expect(libbuildpack.NewYAML().Write(aptFile, content)).To(Succeed())

			Expect(a.Setup()).To(Succeed())

			a.Packages = []string{fooFileUri, barFileUri}
		})

		AfterEach(func() {
			fooServer.Close()
			barServer.Close()
		})

		It("downloads user specified packages using http get's", func() {
			mockCommand.EXPECT().Output(
				"/", "apt-get",
				"-o", "debug::nolocking=true",
				"-o", "dir::cache="+cacheDir+"/apt/cache",
				"-o", "dir::state="+cacheDir+"/apt/state",
				"-o", "dir::etc::sourcelist="+cacheDir+"/apt/sources/sources.list",
				"-o", "dir::etc::trusted="+cacheDir+"/apt/etc/trusted.gpg",
				"-o", "Dir::Etc::preferences="+cacheDir+"/apt/etc/preferences",
				"-y", "--allow-downgrades", "--allow-remove-essential", "--allow-change-held-packages", "-d", "install", "--reinstall",
			).Return("apt output", nil)

			Expect(a.DownloadAll()).To(Succeed())
			Expect(fooServer.ReceivedRequests()).Should(HaveLen(1))
			Expect(barServer.ReceivedRequests()).Should(HaveLen(1))
		})

	})

	Describe("InstallAll", func() {
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
			Expect(a.InstallAll()).To(Succeed())
		})
	})
})
