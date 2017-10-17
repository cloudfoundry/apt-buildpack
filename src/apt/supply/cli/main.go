package main

import (
	"apt/apt"
	"apt/supply"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	logger := libbuildpack.NewLogger(os.Stdout)

	cmd := exec.Command("find", ".")
	cmd.Dir = "/tmp/cache"
	cmd.Stdout = os.Stdout
	cmd.Run()

	buildpackDir, err := libbuildpack.GetBuildpackDir()
	if err != nil {
		logger.Error("Unable to determine buildpack directory: %s", err.Error())
		os.Exit(9)
	}

	manifest, err := libbuildpack.NewManifest(buildpackDir, logger, time.Now())
	if err != nil {
		logger.Error("Unable to load buildpack manifest: %s", err.Error())
		os.Exit(10)
	}

	stager := libbuildpack.NewStager(os.Args[1:], logger, manifest)
	if err := stager.CheckBuildpackValid(); err != nil {
		os.Exit(11)
	}

	if err = stager.SetStagingEnvironment(); err != nil {
		logger.Error("Unable to setup environment variables: %s", err.Error())
		os.Exit(13)
	}

	command := &libbuildpack.Command{}
	a, err := apt.New(command, filepath.Join(stager.BuildDir(), "Aptfile"), stager.CacheDir(), filepath.Join(stager.DepDir(), "apt"))
	if err != nil {
		logger.Error("Unable to initialize apt package: %s", err.Error())
		os.Exit(13)
	}

	supplier := supply.New(stager, a, logger)

	if err := supplier.Run(); err != nil {
		os.Exit(14)
	}

	if err := stager.WriteConfigYml(nil); err != nil {
		logger.Error("Error writing config.yml: %s", err.Error())
		os.Exit(15)
	}

	stager.StagingComplete()
}
