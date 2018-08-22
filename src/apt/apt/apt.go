package apt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Command interface {
	// Execute(string, io.Writer, io.Writer, string, ...string) error
	Output(string, string, ...string) (string, error)
	// Run(*exec.Cmd) error
}

type RepoPriority struct {
	Repository string
	Priority   string
}

type Apt struct {
	command            Command
	options            []string
	aptFilePath        string
	Keys               []string `yaml:"keys"`
	GpgAdvancedOptions []string `yaml:"gpg_advanced_options"`
	Repos              []string `yaml:"repos"`
	Packages           []string `yaml:"packages"`
	Priorities         []RepoPriority `yaml:"priorities"`
	cacheDir           string
	stateDir           string
	sourceList         string
	trustedKeys        string
	installDir         string
	preferences        string
}

func New(command Command, aptFile, cacheDir, installDir string) *Apt {
	sourceList := filepath.Join(cacheDir, "apt", "sources", "sources.list")
	trustedKeys := filepath.Join(cacheDir, "apt", "etc", "trusted.gpg")
	preferences := filepath.Join(cacheDir, "apt", "etc", "preferences")

	return &Apt{
		command:     command,
		aptFilePath: aptFile,
		cacheDir:    filepath.Join(cacheDir, "apt", "cache"),
		stateDir:    filepath.Join(cacheDir, "apt", "state"),
		sourceList:  sourceList,
		trustedKeys: trustedKeys,
		preferences: preferences,
		options: []string{
			"-o", "debug::nolocking=true",
			"-o", "dir::cache=" + filepath.Join(cacheDir, "apt", "cache"),
			"-o", "dir::state=" + filepath.Join(cacheDir, "apt", "state"),
			"-o", "dir::etc::sourcelist=" + sourceList,
			"-o", "dir::etc::trusted=" + trustedKeys,
			"-o", "Dir::Etc::preferences=" + preferences,
		},
		installDir: installDir,
	}
}

func (a *Apt) Setup() error {
	if err := os.MkdirAll(a.cacheDir, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(a.stateDir, 0755); err != nil {
		return err
	}

	if err := libbuildpack.CopyFile("/etc/apt/sources.list", a.sourceList); err != nil {
		return err
	}

	if err := libbuildpack.CopyFile("/etc/apt/trusted.gpg", a.trustedKeys); err != nil {
		return err
	}

	if exists, err := libbuildpack.FileExists("/etc/apt/preferences"); err != nil {
		return err
	} else if exists {
		if err := libbuildpack.CopyFile("/etc/apt/preferences", a.preferences); err != nil {
			return err
		}
	} else {
		dirPath := filepath.Dir(a.preferences)
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			return err
		}
	}

	if err := libbuildpack.NewYAML().Load(a.aptFilePath, a); err != nil {
		return err
	}

	return nil
}

func (a *Apt) HasKeys() bool  { return len(a.Keys) > 0 || len(a.GpgAdvancedOptions) > 0 }
func (a *Apt) HasRepos() bool { return len(a.Repos) > 0 }

func (a *Apt) AddKeys() (string, error) {
	for _, options := range a.GpgAdvancedOptions {
		if out, err := a.command.Output("/", "apt-key", "--keyring", a.trustedKeys, "adv", options); err != nil {
			return out, fmt.Errorf("Could not pass gpg advanced options `%s`: %v", options, err)
		}
	}
	for _, keyURL := range a.Keys {
		if out, err := a.command.Output("/", "apt-key", "--keyring", a.trustedKeys, "adv", "--fetch-keys", keyURL); err != nil {
			return out, fmt.Errorf("Could not add apt key %s: %v", keyURL, err)
		}
	}
	return "", nil
}

func (a *Apt) AddRepos() error {
	f, err := os.OpenFile(a.sourceList, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, repo := range a.Repos {
		if _, err = f.WriteString("\n" + repo); err != nil {
			return err
		}
	}

	if len(a.Priorities) > 0 {
		prefFile, err := os.OpenFile(a.preferences, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer prefFile.Close()

		for _, repoPriority := range a.Priorities {
			if _, err = prefFile.WriteString("\nPackage: *\nPin: release a=" + repoPriority.Repository + "\nPin-Priority: " + repoPriority.Priority + "\n"); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *Apt) Update() (string, error) {
	args := append(a.options, "update")
	return a.command.Output("/", "apt-get", args...)
}

func (a *Apt) Download() (string, error) {
	debPackages := make([]string, 0)
	repoPackages := make([]string, 0)

	for _, pkg := range a.Packages {
		if strings.HasSuffix(pkg, ".deb") {
			debPackages = append(debPackages, pkg)
		} else if pkg != "" {
			repoPackages = append(repoPackages, pkg)
		}
	}

	// download .deb packages individually
	for _, pkg := range debPackages {
		packageFile := filepath.Join(a.cacheDir, "archives", filepath.Base(pkg))
		args := []string{"-s", "-L", "-z", packageFile, "-o", packageFile, pkg}
		if output, err := a.command.Output("/", "curl", args...); err != nil {
			return output, err
		}
	}

	// download all repo packages in one invocation
	aptArgs := append(a.options, "-y", "--force-yes", "-d", "install", "--reinstall")
	args := append(aptArgs, repoPackages...)
	output, err := a.command.Output("/", "apt-get", args...)
	if err != nil {
		return output, err
	}
	fmt.Printf("%s\n", output)

	return "", nil
}

func (a *Apt) Install() (string, error) {
	files, err := filepath.Glob(filepath.Join(a.cacheDir, "archives", "*.deb"))
	if err != nil {
		return "", err
	}

	for _, file := range files {
		fmt.Printf("installing " + filepath.Base(file) + "\n")
		output, err := a.command.Output("/", "dpkg", "-x", file, a.installDir)
		if err != nil {
			fmt.Printf("Error installing packages!\n" + output)
			return output, err
		}
	}
	return "", nil
}
