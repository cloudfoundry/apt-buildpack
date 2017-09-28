package apt

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Command interface {
	// Execute(string, io.Writer, io.Writer, string, ...string) error
	Output(string, string, ...string) (string, error)
	// Run(*exec.Cmd) error
}

type Apt struct {
	command    Command
	options    []string
	aptFile    string
	cacheDir   string
	installDir string
}

func New(command Command, aptFile, cacheDir, installDir string) (*Apt, error) {
	if err := os.MkdirAll(filepath.Join(cacheDir, "apt", "cache"), 0755); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Join(cacheDir, "apt", "state"), 0755); err != nil {
		return nil, err
	}

	return &Apt{
		command:  command,
		aptFile:  aptFile,
		cacheDir: filepath.Join(cacheDir, "apt", "cache"),
		options: []string{
			"-o", "debug::nolocking=true",
			"-o", "dir::cache=" + filepath.Join(cacheDir, "apt", "cache"),
			"-o", "dir::state=" + filepath.Join(cacheDir, "apt", "state"),
		},
		installDir: installDir,
	}, nil
}

func (a *Apt) Update() (string, error) {
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
