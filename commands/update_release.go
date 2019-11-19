package commands

import (
	"fmt"
	"github.com/pivotal-cf/jhanda"
	"github.com/pivotal-cf/kiln/fetcher"
	"github.com/pivotal-cf/kiln/internal/cargo"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/yaml.v2"
	"log"
)

//go:generate counterfeiter -o ./fakes/release_downloader.go --fake-name ReleaseDownloader . ReleaseDownloader
type ReleaseDownloader interface {
	DownloadRelease(releasesDir string, requirement fetcher.ReleaseRequirement) (fetcher.LocalRelease, error)
}

type UpdateRelease struct {
	Options struct {
		Kilnfile       string   `short:"kf" long:"kilnfile" required:"true" description:"path to Kilnfile"`
		Name string `short:"n" long:"name" required:"true" description: "name of release to update""`
		Version string `short:"v" long:"version" required:"true" description: "desired version of release""`
		ReleasesDir string `short:"rd" long:"releases-directory" default:"releases" description:"path to a directory to download releases into"`
	}
	releaseDownloader ReleaseDownloader
	filesystem        billy.Filesystem
	logger            *log.Logger
}

func NewUpdateRelease(logger *log.Logger, filesystem billy.Filesystem,releaseDownloader ReleaseDownloader) UpdateRelease {
	return UpdateRelease{logger: logger, releaseDownloader: releaseDownloader, filesystem: filesystem}
}

func (u UpdateRelease) Execute(args []string) error {
	_, err := jhanda.Parse(&u.Options, args)
	if err != nil {
		panic("banananana")
	}

	releaseRequirement := fetcher.ReleaseRequirement{Name: u.Options.Name, Version: u.Options.Version}

	releaseDir := u.Options.ReleasesDir
	localRelease, err := u.releaseDownloader.DownloadRelease(releaseDir, releaseRequirement)
	if err != nil {
		panic("banana download")
	}

	kilnfileLockPath := fmt.Sprintf("%s.lock", u.Options.Kilnfile)
	var kilnfileLock cargo.KilnfileLock
	kilnfileLockFile, err := u.filesystem.Open(kilnfileLockPath)
	if err != nil {
		panic(err)
	}
	err = yaml.NewDecoder(kilnfileLockFile).Decode(&kilnfileLock)
	if err != nil {
		panic("banana decode")
	}
	err = u.filesystem.Remove(kilnfileLockPath)
	if err != nil {
		panic("banana remove")
	}
	lockFile, err := u.filesystem.Create(kilnfileLockPath)
	if err != nil {
		panic("banana create")
	}

	var matchingRelease *cargo.Release
	for i := range kilnfileLock.Releases {
		if kilnfileLock.Releases[i].Name == u.Options.Name {
			matchingRelease = &kilnfileLock.Releases[i]
			break
		}
	}
	if matchingRelease == nil {
		panic("banana no matching release")
	}
	matchingRelease.Version = u.Options.Version
	sha, err := fetcher.CalculateSum(localRelease.LocalPath(), u.filesystem)
	if err != nil {
		panic(fmt.Sprintf("banana sha1 failed, %+v", err))
	}
	matchingRelease.SHA1 = sha

	updatedLockFileYAML, err := yaml.Marshal(kilnfileLock)
	if err != nil {
		panic("banana marshal")
	}

	_, err = lockFile.Write(updatedLockFileYAML)
	if err != nil {
		panic("banana write")
	}

	u.logger.Printf("Updated %s to %s\n", u.Options.Name, u.Options.Version)
	return nil
}

func (u UpdateRelease) Usage() jhanda.Usage {
	return jhanda.Usage{
		Description:      "",
		ShortDescription: "",
		Flags:            u.Options,
	}
}

type releaseDownloader struct {
	releaseSources []fetcher.ReleaseSource
}

func NewReleaseDownloader(outLogger *log.Logger, kilnfile cargo.Kilnfile) releaseDownloader {
	releaseSources := fetcher.NewReleaseSourcesFactory(outLogger)(kilnfile, false)
	return releaseDownloader{releaseSources: releaseSources}
}

func (rd releaseDownloader) DownloadRelease(releaseDir string, requirement fetcher.ReleaseRequirement) (fetcher.LocalRelease, error) {
	return nil, nil
}
