package app_test

import (
	"testing"

	"github.com/epicchainlabs/epicchain-go/internal/testcli"
	"github.com/epicchainlabs/epicchain-go/internal/versionutil"
	"github.com/epicchainlabs/epicchain-go/pkg/config"
)

func TestCLIVersion(t *testing.T) {
	config.Version = versionutil.TestVersion // Zero-length version string disables '--version' completely.
	e := testcli.NewExecutor(t, false)
	e.Run(t, "neo-go", "--version")
	e.CheckNextLine(t, "^NeoGo")
	e.CheckNextLine(t, "^Version:")
	e.CheckNextLine(t, "^GoVersion:")
	e.CheckEOF(t)
}
