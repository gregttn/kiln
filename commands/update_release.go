package commands

import (
	"fmt"
	"github.com/pivotal-cf/jhanda"
	"github.com/pivotal-cf/kiln/fetcher"
	"github.com/pivotal-cf/kiln/internal/cargo"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/yaml.v2"
)

//go:generate counterfeiter -o ./fakes/release_finder.go --fake-name ReleaseFinder . ReleaseFinder
type ReleaseFinder interface {
	GetMatchedReleases(fetcher.ReleaseRequirementSet, cargo.Stemcell) ([]fetcher.RemoteRelease, error)
	DownloadReleases(releasesDir string, matchedS3Objects []fetcher.RemoteRelease, downloadThreads int) (fetcher.LocalReleaseSet, error)
}

type UpdateRelease struct {
	Options struct {
		Kilnfile       string   `short:"kf" long:"kilnfile" required:"true" description:"path to Kilnfile"`
		Name string `short:"n" long:"name" required:"true" description: "name of release to update""`
		Version string `short:"v" long:"version" required:"true" description: "desired version of release""`
	}
	releaseFinder ReleaseFinder
	filesystem billy.Filesystem
}

func NewUpdateRelease(filesystem billy.Filesystem,releaseFinder ReleaseFinder) UpdateRelease {
	return UpdateRelease{releaseFinder: releaseFinder, filesystem: filesystem}
}

func (u UpdateRelease) Execute(args []string) error {
	// find the release in a remote source
	// remoteRelease, err := releaseFinder.Find(releaseName, releaseVersion)
	releaseRequirementSet := fetcher.ReleaseRequirementSet{}
	releaseRequirementSet[fetcher.ReleaseID{Version: "1.87.8", Name: "capi"}] = fetcher.ReleaseRequirement{Name: "capi", Version: "1.87.8"}

	releaseSet, err := u.releaseFinder.GetMatchedReleases(releaseRequirementSet, cargo.Stemcell{})
	if err != nil {
		panic("banana get matched")
	}

	_, err = u.releaseFinder.DownloadReleases("something",releaseSet, 0)
	if err != nil {
		panic("banana download")
	}

	_, err = jhanda.Parse(&u.Options, args)
	if err != nil {
		panic("banananana")
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

	kilnfileLock.Releases[0].Version = "1.87.8"
	updatedLockFileYAML, err := yaml.Marshal(kilnfileLock)
	if err != nil {
		panic("banana marshal")
	}

	_, err = lockFile.Write(updatedLockFileYAML)
	if err != nil {
		panic("banana write")
	}
	// download it
	// localrelease, err := remoteRelease.download(temporaryPlace)
	// sha 1 sum it
	// shasum(localrelease.path)
	// write the sha1sum and the version number to the kilnfile.lock
	//
	fmt.Println("Updated capi to 1.87.8")
	return nil
}

func (u UpdateRelease) Usage() jhanda.Usage {
	return jhanda.Usage{
		Description:      "",
		ShortDescription: "",
		Flags:            u.Options,
	}
}
