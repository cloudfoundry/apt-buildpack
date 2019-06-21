package supply

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Stager interface {
	LinkDirectoryInDepDir(string, string) error
	DepDir() string
	CacheDir() string
}

type Apt interface {
	Setup() error
	HasKeys() bool
	HasRepos() bool
	AddKeys() error
	AddRepos() error
	Update() error
	DownloadAll() error
	InstallAll() error
	Clean() error
	HasClean() bool
}

type Supplier struct {
	Stager Stager
	Log    *libbuildpack.Logger
	Apt    Apt
}

func New(stager Stager, apt Apt, logger *libbuildpack.Logger) *Supplier {
	return &Supplier{
		Stager: stager,
		Log:    logger,
		Apt:    apt,
	}
}

func (s *Supplier) Run() error {
	if err := s.Apt.Setup(); err != nil {
		s.Log.Error("Failed to setup apt: %v", err)
		return err
	}

	if s.Apt.HasKeys() {
		s.Log.BeginStep("Adding apt keys")
		if err := s.Apt.AddKeys(); err != nil {
			return err
		}
	}

	if s.Apt.HasRepos() {
		s.Log.BeginStep("Adding apt repos")
		if err := s.Apt.AddRepos(); err != nil {
			return err
		}
	}

	if s.Apt.HasClean() {
		s.Log.BeginStep("Cleaning apt cache")
		if err := s.Apt.Clean(); err != nil {
			return err
		}
	}
	s.Log.BeginStep("Updating apt cache")
	if err := s.Apt.Update(); err != nil {
		return err
	}

	s.Log.BeginStep("Downloading apt packages")
	if err := s.Apt.DownloadAll(); err != nil {
		return err
	}

	s.Log.BeginStep("Installing apt packages")
	if err := s.Apt.InstallAll(); err != nil {
		return err
	}

	s.Log.Debug("Creating Symlinks")
	return s.createSymlinks()
}

func (s *Supplier) createSymlinks() error {
	for _, dirs := range [][]string{
		{"usr/bin", "bin"},
		{"usr/lib", "lib"},
		{"usr/lib/i386-linux-gnu", "lib"},
		{"usr/lib/x86_64-linux-gnu", "lib"},
		{"lib/x86_64-linux-gnu", "lib"},
		{"usr/include", "include"},
	} {
		dest := filepath.Join(s.Stager.DepDir(), "apt", dirs[0])
		if exists, err := libbuildpack.FileExists(dest); err != nil {
			return err
		} else if exists {
			if err := s.Stager.LinkDirectoryInDepDir(dest, dirs[1]); err != nil {
				return err
			}
		}
	}

	// copy pkgconfig files instead of linking, then modify
	// the copies to point to the DepDir-based path
	for _, dirs := range [][]string{
		{"usr/lib/i386-linux-gnu/pkgconfig", "pkgconfig"},
		{"usr/lib/x86_64-linux-gnu/pkgconfig", "pkgconfig"},
		{"usr/lib/pkgconfig", "pkgconfig"},
	} {
		dest := filepath.Join(s.Stager.DepDir(), "apt", dirs[0])
		if exists, err := libbuildpack.FileExists(dest); err != nil {
			return err
		} else if exists {
			files, err := ioutil.ReadDir(dest)
			if err != nil {
				return err
			}
			destDir := filepath.Join(s.Stager.DepDir(), dirs[1])
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return err
			}
			for _, file := range files {
				//TODO: better way to copy a file?
				contents, err := ioutil.ReadFile(filepath.Join(dest, file.Name()))
				if err != nil {
					return err
				}
				newContents := strings.Replace(string(contents[:]), "prefix=/usr\n", "prefix="+filepath.Join(s.Stager.DepDir(), "apt", "usr")+"\n", -1)
				err = ioutil.WriteFile(filepath.Join(destDir, file.Name()), []byte(newContents), 0666)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
