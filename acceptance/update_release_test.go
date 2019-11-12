package acceptance_test

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Context("Updating a release to a specific version", func() {
	var kilnfileContents, previousKilnfileLock, kilnfileLockPath, kilnfilePath string
	BeforeEach(func() {
		kilnfileContents = `---
release_sources:
- type: bosh.io
`
		previousKilnfileLock = `---
releases:
- name: "loggregator-agent"
  version: "5.1.0"
  sha1: "a86e10219b0ed9b7b82f0610b7cdc03c13765722"
- name: capi
  sha1: "03ac801323cd23205dde357cc7d2dc9e92bc0c93"
  version: "1.87.0"
stemcell_criteria:
  os: some-os
  version: "4.5.6"
`
		tmpDir, err := ioutil.TempDir("", "kiln-main-test")
		Expect(err).NotTo(HaveOccurred())

		kilnfileLockPath = filepath.Join(tmpDir, "Kilnfile.lock")
		kilnfilePath = filepath.Join(tmpDir, "Kilnfile")
		ioutil.WriteFile(kilnfilePath, []byte(kilnfileContents), 0600)
		ioutil.WriteFile(kilnfileLockPath, []byte(previousKilnfileLock), 0600)
	})

	It("updates the Kilnfile.lock", func() {
		command := exec.Command(pathToMain, "update-release", "--name", "capi", "--version", "1.87.8","--kilnfile",kilnfilePath)
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))
		Expect(session.Out).To(gbytes.Say("Updated capi to 1.87.8"))

		lockContents, err := ioutil.ReadFile(kilnfileLockPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(lockContents)).To(ContainSubstring("1.87.8"))
		Expect(string(lockContents)).To(ContainSubstring("7a7ef183de3252724b6f8e6ca39ad7cf4995fe27"))
		Expect(string(lockContents)).To(Equal(`---
releases:
- name: "loggregator-agent"
  version: "5.1.0"
  sha1: "a86e10219b0ed9b7b82f0610b7cdc03c13765722"
- name: capi
  sha1: "7a7ef183de3252724b6f8e6ca39ad7cf4995fe27"
  version: "1.87.8"
stemcell_criteria:
  os: some-os
  version: "4.5.6"
`))
	})
})
