package fetcher_test

import (
	"github.com/matt-royal/golandreporter"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFetcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithCustomReporters(t, "Fetcher Suite", []Reporter{golandreporter.NewAutoGolandReporter()})
}
