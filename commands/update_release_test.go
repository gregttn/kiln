package commands_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/kiln/commands"
	"github.com/pivotal-cf/kiln/commands/fakes"
	"github.com/pivotal-cf/kiln/fetcher"
	"github.com/pivotal-cf/kiln/internal/cargo"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"path/filepath"
)

var _ = Describe("UpdateRelease", func() {
	var (
		preexistingKilnfileLock string
		kilnfileContents        string
		filesystem              billy.Filesystem
		releaseDownloader       *fakes.ReleaseDownloader
		logger                  *log.Logger
		releasesDir             string
	)

	Context("Execute", func() {
		Context("when updating a dependency to a higher version that exists in the remote", func() {
			var expectedReleaseRequirement fetcher.ReleaseRequirement

			BeforeEach(func() {
				kilnfileContents = `---
release_sources:
- type: bosh.io
`
				preexistingKilnfileLock = `---
releases:
- name: minecraft
  sha1: "developersdevelopersdevelopersdevelopers"
  version: "2.0.1"
- name: capi
  sha1: "03ac801323cd23205dde357cc7d2dc9e92bc0c93"
  version: "1.87.0"
stemcell_criteria:
  os: some-os
  version: "4.5.6"
`
				filesystem = osfs.New("/tmp/")
				kf, _ := filesystem.Create("Kilnfile")
				kf.Write([]byte(kilnfileContents))
				kf.Close()

				kfl, err := filesystem.Create("Kilnfile.lock")
				Expect(err).NotTo(HaveOccurred())
				kfl.Write([]byte(preexistingKilnfileLock))
				kfl.Close()
				releaseDownloader = &fakes.ReleaseDownloader{}

				logger = log.New(GinkgoWriter, "", 0)

				releasesDir = "releases"
				filesystem.MkdirAll(releasesDir, os.ModePerm)
				downloadedReleasePath := filepath.Join(releasesDir, "capi-1.87.8.tgz")
				downloadedReleaseFile, err := filesystem.Create(downloadedReleasePath)
				Expect(err).NotTo(HaveOccurred())
				downloadedReleaseFile.Write([]byte("lots of files"))
				downloadedReleaseFile.Close()

				releaseID := fetcher.ReleaseID{Name: "capi", Version: "1.87.8"}
				expectedDownloadedRelease := fetcher.BuiltRelease{ID: releaseID, Path: downloadedReleasePath}
				releaseDownloader.DownloadReleaseReturns(expectedDownloadedRelease, nil)

				expectedReleaseRequirement = fetcher.ReleaseRequirement{Name: "capi", Version: "1.87.8"}
			})

			It("writes the new version to the kilnfile", func() {
				updateReleaseCommand := commands.NewUpdateRelease(logger, filesystem, releaseDownloader)

				updateReleaseCommand.Execute([]string{
					"--kilnfile", "Kilnfile",
					"--name", "capi",
					"--version", "1.87.8",
					"--releases-directory", releasesDir,
				})

				releaseID := fetcher.ReleaseID{Version: "1.87.8", Name: "capi"}
				expectedReleaseRequirementSet := fetcher.ReleaseRequirementSet{}
				expectedReleaseRequirementSet[releaseID] = fetcher.ReleaseRequirement{Name: "capi", Version: "1.87.8"}

				Expect(releaseDownloader.DownloadReleaseCallCount()).To(Equal(1), "DownloadRelease should be called once and only once")
				downloadDir, releaseRequirement := releaseDownloader.DownloadReleaseArgsForCall(0)
				Expect(downloadDir).To(Equal(releasesDir), "Download directory")
				Expect(releaseRequirement).To(Equal(expectedReleaseRequirement))

				newKilnfileLock, err := filesystem.Open("Kilnfile.lock")
				Expect(err).NotTo(HaveOccurred())

				var kilnfileLock cargo.KilnfileLock
				err = yaml.NewDecoder(newKilnfileLock).Decode(&kilnfileLock)
				Expect(err).NotTo(HaveOccurred())
				Expect(kilnfileLock.Releases).To(HaveLen(2))
				Expect(kilnfileLock.Releases[1].Name).To(Equal("capi"))
				Expect(kilnfileLock.Releases[1].Version).To(Equal("1.87.8"))
				Expect(kilnfileLock.Releases[1].SHA1).To(Equal("ba01716b40a3557d699d024d76c307e351e96829"))
			})
		})
	})
})
