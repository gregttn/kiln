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
	"reflect"
)

var _ = Describe("UpdateRelease", func() {
	var preexistingKilnfileLock string
	var kilnfileContents string
	var filesystem billy.Filesystem
	var releaseFinder *fakes.ReleaseFinder
	Context("Execute", func() {
		Context("when updating a dependency to a higher version that exists in the remote", func() {
			var expectedRemoteReleaseSet []fetcher.RemoteRelease
			BeforeEach(func() {
				kilnfileContents = `---
release_sources:
- type: bosh.io
`
				preexistingKilnfileLock = `---
releases:
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

				kfl, _ := filesystem.Create("Kilnfile.lock")
				kfl.Write([]byte(preexistingKilnfileLock))
				kfl.Close()
				releaseFinder = &fakes.ReleaseFinder{}

				expectedRemoteReleaseSet = []fetcher.RemoteRelease{fetcher.BuiltRelease{ID: fetcher.ReleaseID{Name: "test", Version: "also test"}, Path: "data"}}
				releaseFinder.GetMatchedReleasesReturns(expectedRemoteReleaseSet, nil)

			})
 			It("writes the new version to the kilnfile", func() {
				updateReleaseCommand := commands.NewUpdateRelease(filesystem, releaseFinder)

				updateReleaseCommand.Execute([]string{"--kilnfile", "Kilnfile", "--name", "capi", "--version", "1.87.8"})

				releaseID := fetcher.ReleaseID{Version: "1.87.8", Name: "capi"}
				expectedReleaseRequirementSet := fetcher.ReleaseRequirementSet{}
				expectedReleaseRequirementSet[releaseID] = fetcher.ReleaseRequirement{Name: "capi", Version: "1.87.8"}

				Expect(releaseFinder.GetMatchedReleasesCallCount()).To(Equal(1), "GetMatchedReleases should be called once and only once")
				receivedReleaseRequirementSet, _ := releaseFinder.GetMatchedReleasesArgsForCall(0)
				Expect(reflect.DeepEqual(expectedReleaseRequirementSet, receivedReleaseRequirementSet)).To(BeTrue())

				Expect(releaseFinder.DownloadReleasesCallCount()).To(Equal(1), "DownloadReleases should be called once and only once")
				downloadDir, receivedRemoteReleaseSet, threads := releaseFinder.DownloadReleasesArgsForCall(0)
				Expect(downloadDir).To(Equal("something"), "Download directory")
				Expect(threads).To(Equal(0))

				Expect(receivedRemoteReleaseSet).To(Equal(expectedRemoteReleaseSet))


				newKilnfileLock, err := filesystem.Open("Kilnfile.lock")
				Expect(err).NotTo(HaveOccurred())

				var kilnfileLock cargo.KilnfileLock
				err = yaml.NewDecoder(newKilnfileLock).Decode(&kilnfileLock)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(kilnfileLock.Releases)).To(Equal(1))
				Expect(kilnfileLock.Releases[0].Name).To(Equal("capi"))
				Expect(kilnfileLock.Releases[0].Version).To(Equal("1.87.8"))
				Expect(kilnfileLock.Releases[0].SHA1).To(Equal("7a7ef183de3252724b6f8e6ca39ad7cf4995fe27"))
			})
		})
	})
})
