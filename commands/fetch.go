package commands

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/pivotal-cf/kiln/fetcher"

	"github.com/pivotal-cf/jhanda"
	"github.com/pivotal-cf/kiln/builder"
	"github.com/pivotal-cf/kiln/internal/baking"
	"github.com/pivotal-cf/kiln/internal/cargo"
	"gopkg.in/yaml.v2"
)

type ConfigFileError struct {
	HumanReadableConfigFileName string
	err                         error
}

func (err ConfigFileError) Unwrap() error {
	return err.err
}

func (err ConfigFileError) Error() string {
	return fmt.Sprintf("encountered a configuration file error with %s: %s", err.HumanReadableConfigFileName, err.err.Error())
}

type ErrorMissingReleases fetcher.ReleaseRequirementSet

func (releases ErrorMissingReleases) Error() string {
	var missing []string
	for id, _ := range releases {
		missing = append(missing, fmt.Sprintf("- %s (%s)", id.Name, id.Version))
	}
	return fmt.Sprintf("could not find the following releases\n%s", strings.Join(missing, "\n"))
}

type Fetch struct {
	logger *log.Logger

	releaseSourcesFactory ReleaseSourcesFactory
	localReleaseDirectory LocalReleaseDirectory

	Options struct {
		Kilnfile    string `short:"kf" long:"kilnfile" default:"Kilnfile" description:"path to Kilnfile"`
		ReleasesDir string `short:"rd" long:"releases-directory" default:"releases" description:"path to a directory to download releases into"`

		VariablesFiles               []string `short:"vf" long:"variables-file" description:"path to variables file"`
		Variables                    []string `short:"vr" long:"variable" description:"variable in key=value format"`
		DownloadThreads              int      `short:"dt" long:"download-threads" description:"number of parallel threads to download parts from S3"`
		NoConfirm                    bool     `short:"n" long:"no-confirm" description:"non-interactive mode, will delete extra releases in releases dir without prompting"`
		AllowOnlyPublishableReleases bool     `long:"allow-only-publishable-releases" description:"include releases that would not be shipped with the tile (development builds)"`
	}
}

//go:generate counterfeiter -o ./fakes/release_sources_factory.go --fake-name ReleaseSourcesFactory . ReleaseSourcesFactory
type ReleaseSourcesFactory interface {
	ReleaseSources(cargo.Kilnfile, bool) []fetcher.ReleaseSource
}

func NewFetch(logger *log.Logger, releaseSourcesFactory ReleaseSourcesFactory, localReleaseDirectory LocalReleaseDirectory) Fetch {
	return Fetch{
		logger:                logger,
		localReleaseDirectory: localReleaseDirectory,
		releaseSourcesFactory: releaseSourcesFactory,
	}
}

//go:generate counterfeiter -o ./fakes/local_release_directory.go --fake-name LocalReleaseDirectory . LocalReleaseDirectory
type LocalReleaseDirectory interface {
	GetLocalReleases(releasesDir string) (fetcher.LocalReleaseSet, error)
	DeleteExtraReleases(extraReleases fetcher.LocalReleaseSet, noConfirm bool) error
	VerifyChecksums(downloadedReleases fetcher.LocalReleaseSet, kilnfileLock cargo.KilnfileLock) error
}

func (f Fetch) Execute(args []string) error {
	kilnfile, kilnfileLock, availableLocalReleaseSet, err := f.setup(args)
	if err != nil {
		return err
	}

	desiredReleaseSet := fetcher.NewReleaseRequirementSet(kilnfileLock)
	satisfiedReleaseSet, unsatisfiedReleaseSet, extraReleaseSet := desiredReleaseSet.Partition(availableLocalReleaseSet)

	err = f.localReleaseDirectory.DeleteExtraReleases(extraReleaseSet, f.Options.NoConfirm)
	if err != nil {
		f.logger.Println("failed deleting some releases: ", err.Error())
	}

	if len(unsatisfiedReleaseSet) > 0 {
		f.logger.Printf("Found %d missing releases to download", len(unsatisfiedReleaseSet))

		satisfiedReleaseSet, unsatisfiedReleaseSet, err = f.downloadMissingReleases(kilnfile, satisfiedReleaseSet, unsatisfiedReleaseSet, kilnfileLock.Stemcell)
		if err != nil {
			return err
		}
	}

	if len(unsatisfiedReleaseSet) > 0 {
		return ErrorMissingReleases(unsatisfiedReleaseSet)
	}

	return f.localReleaseDirectory.VerifyChecksums(satisfiedReleaseSet, kilnfileLock)
}

func (f *Fetch) setup(args []string) (cargo.Kilnfile, cargo.KilnfileLock, fetcher.LocalReleaseSet, error) {
	args, err := jhanda.Parse(&f.Options, args)

	if err != nil {
		return cargo.Kilnfile{}, cargo.KilnfileLock{}, nil, err
	}
	if !f.Options.AllowOnlyPublishableReleases {
		f.logger.Println("WARNING - the \"allow-only-publishable-releases\" flag was not set. Some fetched releases may be intended for development/testing only.\nEXERCISE CAUTION WHEN PUBLISHING A TILE WITH THESE RELEASES!")
	}
	if _, err := os.Stat(f.Options.ReleasesDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(f.Options.ReleasesDir, 0777)
		} else {
			return cargo.Kilnfile{}, cargo.KilnfileLock{}, nil, fmt.Errorf("error with releases directory %s: %s", f.Options.ReleasesDir, err)
		}
	}
	templateVariablesService := baking.NewTemplateVariablesService()
	templateVariables, err := templateVariablesService.FromPathsAndPairs(f.Options.VariablesFiles, f.Options.Variables)
	if err != nil {
		return cargo.Kilnfile{}, cargo.KilnfileLock{}, nil, fmt.Errorf("failed to parse template variables: %s", err)
	}

	kilnfileYAML, err := ioutil.ReadFile(f.Options.Kilnfile)
	if err != nil {
		return cargo.Kilnfile{}, cargo.KilnfileLock{}, nil, err
	}
	interpolator := builder.NewInterpolator()
	interpolatedMetadata, err := interpolator.Interpolate(builder.InterpolateInput{
		Variables: templateVariables,
	}, kilnfileYAML)
	if err != nil {
		return cargo.Kilnfile{}, cargo.KilnfileLock{}, nil, ConfigFileError{err: err, HumanReadableConfigFileName: "interpolating variable files with Kilnfile"}
	}

	f.logger.Println("getting release information from " + f.Options.Kilnfile)
	var kilnfile cargo.Kilnfile
	err = yaml.Unmarshal(interpolatedMetadata, &kilnfile)
	if err != nil {
		return cargo.Kilnfile{}, cargo.KilnfileLock{}, nil, ConfigFileError{err: err, HumanReadableConfigFileName: "Kilnfile specification " + f.Options.Kilnfile}
	}

	f.logger.Println("getting release information from Kilnfile.lock")
	lockFileName := fmt.Sprintf("%s.lock", f.Options.Kilnfile)
	lockFile, err := os.Open(lockFileName)
	if err != nil {
		return cargo.Kilnfile{}, cargo.KilnfileLock{}, nil, err
	}
	defer lockFile.Close()

	var kilnfileLock cargo.KilnfileLock
	err = yaml.NewDecoder(lockFile).Decode(&kilnfileLock)
	if err != nil {
		return cargo.Kilnfile{}, cargo.KilnfileLock{}, nil, ConfigFileError{err: err, HumanReadableConfigFileName: "Kilnfile.lock " + lockFileName}
	}

	availableLocalReleaseSet, err := f.localReleaseDirectory.GetLocalReleases(f.Options.ReleasesDir)
	if err != nil {
		return cargo.Kilnfile{}, cargo.KilnfileLock{}, nil, err
	}

	return kilnfile, kilnfileLock, availableLocalReleaseSet, nil
}

func (f Fetch) downloadMissingReleases(kilnfile cargo.Kilnfile, satisfiedReleaseSet fetcher.LocalReleaseSet, unsatisfiedReleaseSet fetcher.ReleaseRequirementSet, stemcell cargo.Stemcell) (satisfied fetcher.LocalReleaseSet, unsatisfied fetcher.ReleaseRequirementSet, err error) {
	releaseSources := f.releaseSourcesFactory.ReleaseSources(kilnfile, f.Options.AllowOnlyPublishableReleases)
	for _, releaseSource := range releaseSources {
		if len(unsatisfiedReleaseSet) == 0 {
			break
		}
		remoteReleases, err := releaseSource.GetMatchedReleases(unsatisfiedReleaseSet, stemcell)
		if err != nil {
			return nil, nil, err
		}

		localReleases, err := releaseSource.DownloadReleases(f.Options.ReleasesDir, remoteReleases, f.Options.DownloadThreads)
		if err != nil {
			return nil, nil, err
		}

		satisfiedReleaseSet = satisfiedReleaseSet.With(localReleases)
		unsatisfiedReleaseSet = unsatisfiedReleaseSet.WithoutReleases(localReleases.ReleaseIDs())
	}

	return satisfiedReleaseSet, unsatisfiedReleaseSet, nil
}

func (f Fetch) Usage() jhanda.Usage {
	return jhanda.Usage{
		Description:      "Fetches releases listed in Kilnfile.lock from S3 and downloads it locally",
		ShortDescription: "fetches releases",
		Flags:            f.Options,
	}
}
