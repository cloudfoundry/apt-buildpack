package apt

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/libbuildpack"
)

type Command interface {
	Output(string, string, ...string) (string, error)
	Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error
}

type Repository struct {
	Name     string
	Priority string
}

func (r *Repository) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err == nil {
		r.Name = name
		return nil
	}

	data := struct {
		Name     string
		Priority string
	}{}
	err := unmarshal(&data)
	if err != nil {
		return err
	}

	r.Name = data.Name
	r.Priority = data.Priority
	return nil
}

type Apt struct {
	command            Command
	options            []string
	aptFilePath        string
	TruncateSources    bool         `yaml:"truncatesources,omitempty"`
	CleanCache         bool         `yaml:"cleancache,omitempty"`
	Keys               []string     `yaml:"keys"`
	GpgAdvancedOptions []string     `yaml:"gpg_advanced_options"`
	Repos              []Repository `yaml:"repos"`
	Packages           []string     `yaml:"packages"`
	rootDir            string
	cacheDir           string
	stateDir           string
	sourceList         string
	trustedKeys        string
	installDir         string
	preferences        string
	archiveDir         string
}

func New(command Command, aptFile, rootDir, cacheDir, installDir string) *Apt {
	sourceList := filepath.Join(cacheDir, "apt", "sources", "sources.list")
	trustedKeys := filepath.Join(cacheDir, "apt", "etc", "trusted.gpg")
	preferences := filepath.Join(cacheDir, "apt", "etc", "preferences")
	aptCacheDir := filepath.Join(cacheDir, "apt", "cache")
	stateDir := filepath.Join(cacheDir, "apt", "state")

	return &Apt{
		command:     command,
		aptFilePath: aptFile,
		rootDir:     rootDir,
		cacheDir:    aptCacheDir,
		stateDir:    stateDir,
		sourceList:  sourceList,
		trustedKeys: trustedKeys,
		preferences: preferences,
		options: []string{
			"-o", "debug::nolocking=true",
			"-o", "dir::cache=" + aptCacheDir,
			"-o", "dir::state=" + stateDir,
			"-o", "dir::etc::sourcelist=" + sourceList,
			"-o", "dir::etc::trusted=" + trustedKeys,
			"-o", "Dir::Etc::preferences=" + preferences,
		},
		installDir: installDir,
		archiveDir: filepath.Join(aptCacheDir, "archives"),
	}
}

func (a *Apt) Setup() error {
	if err := os.MkdirAll(a.cacheDir, os.ModePerm); err != nil {
		return err
	}

	if err := os.MkdirAll(a.archiveDir, os.ModePerm); err != nil {
		return err
	}

	if err := os.MkdirAll(a.stateDir, os.ModePerm); err != nil {
		return err
	}

	aptSources := filepath.Join(a.rootDir, "sources.list")
	if err := libbuildpack.CopyFile(aptSources, a.sourceList); err != nil {
		return err
	}

	aptGPG := filepath.Join(a.rootDir, "trusted.gpg")
	if exists, err := libbuildpack.FileExists(aptGPG); err != nil {
		return err
	} else if exists {
		if err := libbuildpack.CopyFile(aptGPG, a.trustedKeys); err != nil {
			return err
		}
	}

	aptPrefs := filepath.Join(a.rootDir, "preferences")
	if exists, err := libbuildpack.FileExists(aptPrefs); err != nil {
		return err
	} else if exists {
		if err := libbuildpack.CopyFile(aptPrefs, a.preferences); err != nil {
			return err
		}
	} else {
		dirPath := filepath.Dir(a.preferences)
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			return err
		}
	}

	return libbuildpack.NewYAML().Load(a.aptFilePath, a)
}

func (a *Apt) HasKeys() bool {
	return len(a.Keys) > 0 || len(a.GpgAdvancedOptions) > 0
}

func (a *Apt) HasRepos() bool {
	return len(a.Repos) > 0
}

func (a *Apt) AddKeys() error {
	for _, options := range a.GpgAdvancedOptions {
		if out, err := a.command.Output("/", "apt-key", "--keyring", a.trustedKeys, "adv", options); err != nil {
			return fmt.Errorf("could not pass gpg advanced options %s\n\n%s\n\n%s", options, out, err)
		}
	}

	for _, keyURL := range a.Keys {
		if out, err := a.command.Output("/", "apt-key", "--keyring", a.trustedKeys, "adv", "--fetch-keys", keyURL); err != nil {
			return fmt.Errorf("could not add apt key %s\n\n%s\n\n%s", keyURL, out, err)
		}
	}

	return nil
}

func (a *Apt) AddRepos() error {
	openmode := os.O_APPEND

	if a.TruncateSources {
		openmode = os.O_TRUNC
		fmt.Print("Truncating sources.list file.\n")
	}

	f, err := os.OpenFile(a.sourceList, openmode|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, repo := range a.Repos {
		if _, err = f.WriteString("\n" + repo.Name); err != nil {
			return err
		}
		fmt.Printf("Added repo %v\n", repo)
	}

	prefFile, err := os.OpenFile(a.preferences, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer prefFile.Close()

	for _, repo := range a.Repos {
		if repo.Priority != "" {
			if _, err = prefFile.WriteString("\nPackage: *\nPin: release a=" + repo.Name + "\nPin-Priority: " + repo.Priority + "\n"); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *Apt) HasClean() bool {
	return a.CleanCache
}

func (a *Apt) Clean() error {
	fmt.Printf("Cleaning apt cache \n")
	args := append(a.options, "clean")
	if out, err := a.command.Output("/", "apt-get", args...); err != nil {
		fmt.Printf("Info: error running apt-get clean %s\n\n%s", out, err)
	}
	args2 := append(a.options, "autoclean")
	if out, err := a.command.Output("/", "apt-get", args2...); err != nil {
		fmt.Printf("Info: error running apt-get autoclean %s\n\n%s", out, err)
	}

	return nil
}

func (a *Apt) Update() error {
	args := append(a.options, "update")

	var errBuff bytes.Buffer
	if err := a.command.Execute("/", &errBuff, &errBuff, "apt-get", args...); err != nil {
		return fmt.Errorf("failed to apt-get update %s\n\n%s", errBuff.String(), err)
	}
	return nil
}

func (a *Apt) DownloadAll() error {
	debPackages, repoPackages := make([]string, 0), make([]string, 0)

	for _, pkg := range a.Packages {
		if strings.HasSuffix(pkg, ".deb") {
			debPackages = append(debPackages, pkg)
		} else if pkg != "" {
			repoPackages = append(repoPackages, pkg)
		}
	}

	for _, pkg := range debPackages {
		err := a.download(pkg)
		if err != nil {
			return err
		}
	}

	// download all repo packages in one invocation
	aptArgs := append(a.options, "-y", "--allow-downgrades", "--allow-remove-essential", "--allow-change-held-packages", "-d", "install", "--reinstall")
	args := append(aptArgs, repoPackages...)
	out, err := a.command.Output("/", "apt-get", args...)
	if err != nil {
		return fmt.Errorf("failed apt-get install %s\n\n%s", out, err)
	}

	return nil
}

func (a *Apt) InstallAll() error {
	files, err := filepath.Glob(filepath.Join(a.archiveDir, "*.deb"))
	if err != nil {
		return err
	}

	for _, file := range files {
		err := a.install(filepath.Base(file))
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Apt) install(pkg string) error {
	output, err := a.command.Output("/", "dpkg", "-x", filepath.Join(a.archiveDir, pkg), a.installDir)
	if err != nil {
		return fmt.Errorf("failed to install pkg %s\n\n%s\n\n%s", pkg, output, err.Error())
	}
	return nil
}

func (a *Apt) download(pkg string) error {
	var lastModLocal time.Time

	downloadedPkg := filepath.Join(a.archiveDir, filepath.Base(pkg))
	exists, err := libbuildpack.FileExists(downloadedPkg)
	if err != nil {
		return err
	}

	packageFile, err := os.OpenFile(downloadedPkg, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer packageFile.Close()

	if exists {
		localFileStat, err := packageFile.Stat()
		if err != nil {
			return err
		}
		lastModLocal = localFileStat.ModTime()
	} else {
		lastModLocal = time.Time{}
	}

	resp, err := http.Get(pkg)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	lastModRemote, err := http.ParseTime(resp.Header.Get("last-modified"))
	if err != nil {
		// handle ParseTime error on invalid last-modified headers
		if _, ok := err.(*time.ParseError); ok {
			lastModRemote = time.Now()
		} else {
			return err
		}
	}

	diff := lastModRemote.Sub(lastModLocal)
	if diff >= 0 {
		if n, err := io.Copy(packageFile, resp.Body); err != nil {
			return err
		} else if n < resp.ContentLength {
			return fmt.Errorf("could only write %d bytes of total %d for pkg %s", n, resp.ContentLength, packageFile.Name())
		}
	}

	return nil
}
