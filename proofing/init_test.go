package proofing_test

import (
	"github.com/matt-royal/golandreporter"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestProofing(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithCustomReporters(t, "proofing", []Reporter{golandreporter.NewAutoGolandReporter()})
}
