package commands_test

import (
	"github.com/matt-royal/golandreporter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/jhanda"

	"testing"
)

func TestCommands(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithCustomReporters(t, "commands", []Reporter{golandreporter.NewAutoGolandReporter()})
}

//go:generate counterfeiter -o ./fakes/command.go --fake-name Command . command
type command interface {
	jhanda.Command
}
