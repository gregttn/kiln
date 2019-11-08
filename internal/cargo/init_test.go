package cargo_test

import (
	"github.com/matt-royal/golandreporter"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCargo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithCustomReporters(t, "internal/cargo", []Reporter{golandreporter.NewAutoGolandReporter()})
}
