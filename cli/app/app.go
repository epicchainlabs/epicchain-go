package app

import (
	"fmt"
	"os"
	"runtime"

	"github.com/epicchainlabs/epicchain-go/cli/query"
	"github.com/epicchainlabs/epicchain-go/cli/server"
	"github.com/epicchainlabs/epicchain-go/cli/smartcontract"
	"github.com/epicchainlabs/epicchain-go/cli/util"
	"github.com/epicchainlabs/epicchain-go/cli/vm"
	"github.com/epicchainlabs/epicchain-go/cli/wallet"
	"github.com/epicchainlabs/epicchain-go/pkg/config"
	"github.com/urfave/cli"
)

func versionPrinter(c *cli.Context) {
	_, _ = fmt.Fprintf(c.App.Writer, "NeoGo\nVersion: %s\nGoVersion: %s\n",
		config.Version,
		runtime.Version(),
	)
}

// New creates a NeoGo instance of [cli.App] with all commands included.
func New() *cli.App {
	cli.VersionPrinter = versionPrinter
	ctl := cli.NewApp()
	ctl.Name = "neo-go"
	ctl.Version = config.Version
	ctl.Usage = "Official Go client for Neo"
	ctl.ErrWriter = os.Stdout

	ctl.Commands = append(ctl.Commands, server.NewCommands()...)
	ctl.Commands = append(ctl.Commands, smartcontract.NewCommands()...)
	ctl.Commands = append(ctl.Commands, wallet.NewCommands()...)
	ctl.Commands = append(ctl.Commands, vm.NewCommands()...)
	ctl.Commands = append(ctl.Commands, util.NewCommands()...)
	ctl.Commands = append(ctl.Commands, query.NewCommands()...)
	return ctl
}
