package supply

import (
	"io/ioutil"
	"path/filepath"
	"os"
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
	AddKeys() (string, error)
	AddRepos() error
	Update() (string, error)
	Download() (string, error)
	Install() (string, error)
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
		if output, err := s.Apt.AddKeys(); err != nil {
			s.Log.Error("Failed to add apt keys: %v", err)
			s.Log.Info(output)
			return err
		}
	}

	if s.Apt.HasRepos() {
		s.Log.BeginStep("Adding apt repos")
		if err := s.Apt.AddRepos(); err != nil {
			s.Log.Error("Failed to add apt repos: %v", err)
			return err
		}
	}

	s.Log.BeginStep("Updating apt cache")
	if output, err := s.Apt.Update(); err != nil {
		s.Log.Error("Failed to update apt cache: %v", err)
		s.Log.Info(output)
		return err
	}

	s.Log.BeginStep("Downloading apt packages")
	if output, err := s.Apt.Download(); err != nil {
		s.Log.Error("Failed to download apt packages: %v", err)
		s.Log.Info(output)
		return err
	}

	s.Log.BeginStep("Installing apt packages")
	if output, err := s.Apt.Install(); err != nil {
		s.Log.Error("Failed to install apt packages: %v", err)
		s.Log.Info(output)
		return err
	}

	s.Log.Debug("Creating Symlinks")
	if err := s.createSymlinks(); err != nil {
		s.Log.Error("Could not link files: %v", err)
		return err
	}

	return nil
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
				newContents := strings.Replace(string(contents[:]), "prefix=/usr\n", "prefix=" + filepath.Join(s.Stager.DepDir(), "apt", "usr") + "\n", -1)
				err = ioutil.WriteFile(filepath.Join(destDir, file.Name()), []byte(newContents), 0644)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
