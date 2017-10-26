package apt

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Command interface {
	// Execute(string, io.Writer, io.Writer, string, ...string) error
	Output(string, string, ...string) (string, error)
	// Run(*exec.Cmd) error
}

type Apt struct {
	command     Command
	options     []string
	aptFile     string
	cacheDir    string
	sourceList  string
	trustedKeys string
	installDir  string
}

func New(command Command, aptFile, cacheDir, installDir string) (*Apt, error) {
	if err := os.MkdirAll(filepath.Join(cacheDir, "apt", "cache"), 0755); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Join(cacheDir, "apt", "state"), 0755); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Join(cacheDir, "apt", "sources"), 0755); err != nil {
		return nil, err
	}

	sourceList := filepath.Join(cacheDir, "apt", "sources", "sources.list")
	libbuildpack.CopyFile("/etc/apt/sources.list", sourceList)

	trustedKeys := filepath.Join(cacheDir, "apt", "etc", "trusted.gpg")
	libbuildpack.CopyFile("/etc/apt/trusted.gpg", trustedKeys)

	return &Apt{
		command:     command,
		aptFile:     aptFile,
		cacheDir:    filepath.Join(cacheDir, "apt", "cache"),
		sourceList:  sourceList,
		trustedKeys: trustedKeys,
		options: []string{
			"-o", "debug::nolocking=true",
			"-o", "dir::cache=" + filepath.Join(cacheDir, "apt", "cache"),
			"-o", "dir::state=" + filepath.Join(cacheDir, "apt", "state"),
			"-o", "dir::etc::sourcelist=" + sourceList,
			"-o", "dir::etc::trusted=" + trustedKeys,
		},
		installDir: installDir,
	}, nil
}

func (a *Apt) Update() (string, error) {
	text, err := ioutil.ReadFile(a.aptFile)
	if err != nil {
		return "", err
	}

	keyRE, _ := regexp.Compile("^:key:(.*)$")
	for _, pkg := range strings.Split(string(text), "\n") {
		keyMatch := keyRE.FindStringSubmatch(pkg)
		if len(keyMatch) == 2 {
			keyURL := keyMatch[1]
			fmt.Println("Installing custom repository key from", keyURL)
			out, err := a.command.Output("/", "apt-key", "--keyring", a.trustedKeys, "adv", "--fetch-keys", keyURL)
			if err != nil {
				return out, err
			}
		}
	}

	repoRE, _ := regexp.Compile("^:repo:(.*)$")
	for _, pkg := range strings.Split(string(text), "\n") {
		repoMatch := repoRE.FindStringSubmatch(pkg)
		if len(repoMatch) == 2 {
			repositoryStr := repoMatch[1]

			f, err := os.OpenFile(a.sourceList, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				return "", err
			}

			defer f.Close()

			if _, err = f.WriteString(repositoryStr); err != nil {
				return "", err
			}
		}
	}
	args := append(a.options, "update")
	return a.command.Output("/", "apt-get", args...)
}

func (a *Apt) Download() (string, error) {
	aptArgs := append(a.options, "-y", "--force-yes", "-d", "install", "--reinstall")

	text, err := ioutil.ReadFile(a.aptFile)
	if err != nil {
		return "", err
	}

	for _, pkg := range strings.Split(string(text), "\n") {
		if strings.HasSuffix(pkg, ".deb") {
			packageFile := filepath.Join(a.cacheDir, "archives", filepath.Base(pkg))
			args := []string{"-s", "-L", "-z", packageFile, "-o", packageFile, pkg}
			if output, err := a.command.Output("/", "curl", args...); err != nil {
				return output, err
			}

		} else if strings.HasPrefix(pkg, ":") {
			// Skip. This line was used earlier to add custom repository

		} else if pkg != "" {
			args := append(aptArgs, pkg)
			if output, err := a.command.Output("/", "apt-get", args...); err != nil {
				return output, err
			}
		}
	}

	return "", nil
}

func (a *Apt) Install() (string, error) {
	files, err := filepath.Glob(filepath.Join(a.cacheDir, "archives", "*.deb"))
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if output, err := a.command.Output("/", "dpkg", "-x", file, a.installDir); err != nil {
			return output, err
		}
	}
	return "", nil
}
