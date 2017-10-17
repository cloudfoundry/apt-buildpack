package supply

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Stager interface {
	LinkDirectoryInDepDir(string, string) error
	DepDir() string
	CacheDir() string
}

type Apt interface {
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
		{"usr/lib/i386-linux-gnu/pkgconfig", "pkgconfig"},
		{"usr/lib/x86_64-linux-gnu/pkgconfig", "pkgconfig"},
		{"usr/lib/pkgconfig", "pkgconfig"},
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
	return nil
}
