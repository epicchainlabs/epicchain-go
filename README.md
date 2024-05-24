
<p align="center">
  <b>Go</b> Node and SDK for the <a href="https://epicchain.org">epicchain</a> blockchain.
</p>

<hr />

[![codecov](https://codecov.io/gh/nspcc-dev/epicchain-go/branch/master/graph/badge.svg)](https://codecov.io/gh/nspcc-dev/epicchain-go)
[![GithubWorkflows Tests](https://github.com/nspcc-dev/epicchain-go/actions/workflows/tests.yml/badge.svg)](https://github.com/nspcc-dev/epicchain-go/actions/workflows/tests.yml)
[![Report](https://goreportcard.com/badge/github.com/nspcc-dev/epicchain-go)](https://goreportcard.com/report/github.com/nspcc-dev/epicchain-go)
[![GoDoc](https://godoc.org/github.com/nspcc-dev/epicchain-go?status.svg)](https://godoc.org/github.com/nspcc-dev/epicchain-go)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/nspcc-dev/epicchain-go?sort=semver)
![License](https://img.shields.io/github/license/nspcc-dev/epicchain-go.svg?style=popout)

# Overview

epicchainGo is a complete platform for distributed application development built on
top of and compatible with the [epicchain project](https://github.com/epicchain-project).
This includes, but is not limited to (see documentation for more details):

- [Consensus node](docs/consensus.md)
- [RPC node & client](docs/rpc.md)
- [CLI tool](docs/cli.md)
- [Smart contract compiler](docs/compiler.md)
- [epicchain virtual machine](docs/vm.md)
- [Smart contract examples](examples/README.md)
- [Oracle service](docs/oracle.md)
- [State validation service](docs/stateroots.md)

The protocol implemented here is epicchain N3-compatible. However, you can also find
an implementation of the epicchain Legacy protocol in the [**master-2.x**
branch](https://github.com/nspcc-dev/epicchain-go/tree/master-2.x) and releases
before 0.80.0 (**0.7X.Y** track).

# Getting started

## Installation

epicchainGo is distributed as a single binary that includes all the functionality
provided (but smart contract compiler requires Go compiler to operate). You
can grab it from the [releases page](https://github.com/nspcc-dev/epicchain-go/releases), use a Docker image (see
[Docker Hub](https://hub.docker.com/r/nspccdev/epicchain-go) for various releases of
epicchainGo, `:latest` points to the latest release) or build it yourself.

### Building

Building epicchainGo requires Go 1.20+ and `make`:

```sh
make
```

The resulting binary is `bin/epicchain-go`. Note that using some random revision
from the `master` branch is not recommended (it can have any number of
incompatibilities and bugs depending on the development stage); please use
tagged released versions.

#### Building on Windows

To build epicchainGo on the Windows platform, we recommend you to install `make` from the [MinGW
package](https://osdn.net/projects/mingw/). Then, you can build epicchainGo with:

```sh
make
```

The resulting binary is `bin/epicchain-go.exe`.

## Running a node

A node needs to connect to some network, either a local one (usually referred to
as `privnet`) or public (like `mainnet` or `testnet`). Network configuration
is stored in a file, and epicchainGo allows you to store multiple files in one
directory (`./config` by default) and easily switch between them using network
flags.

To start an epicchain node on a private network, use:

```sh
./bin/epicchain-go node
```

Or specify a different network with an appropriate flag like this:

```sh
./bin/epicchain-go node --mainnet
```

Available network flags:
- `--mainnet, -m`
- `--privnet, -p`
- `--testnet, -t`

To run a consensus/committee node, refer to [consensus
documentation](docs/consensus.md).

If you're running a node on Windows, please turn off or configure Windows
Firewall appropriately (allowing inbound connections to the P2P port).

### Docker

By default, the `CMD` is set to run a node on `privnet`. So, to do this, simply run:

```bash
docker run -d --name epicchain-go -p 20332:20332 -p 20331:20331 nspccdev/epicchain-go
```

This will start a node on `privnet` and expose the node's ports `20332` (P2P
protocol) and `20331` (JSON-RPC server).

### Importing mainnet/testnet dump files

If you want to jump-start your mainnet or testnet node with [chain archives
provided by NGD](https://sync.ngd.network/), follow these instructions:
```
$ wget .../chain.acc.zip # chain dump file
$ unzip chain.acc.zip
$ ./bin/epicchain-go db restore -m -i chain.acc # for testnet use '-t' flag instead of '-m'
```

The process differs from the C# node in that block importing is a separate
mode. After it ends, the node can be started normally.

## Running a private network

Refer to [consensus node documentation](docs/consensus.md).

## Smart contract development

Please refer to the [epicchainGo smart contract development
workshop](https://github.com/nspcc-dev/epicchain-go-sc-wrkshp), which shows some
simple contracts that can be compiled/deployed/run using the epicchainGo compiler, SDK,
and a private network. For details on how Go code is translated to epicchain VM
bytecode and what you can and cannot do in a smart contract, please refer to the
[compiler documentation](docs/compiler.md).

Refer to [examples](examples/README.md) for more epicchain smart contract examples
written in Go.

## Wallets

epicchainGo wallet is just an
[NEP-6](https://github.com/epicchain-project/proposals/blob/68398d28b6932b8dd2b377d5d51bca7b0442f532/nep-6.mediawiki)
file that is used by CLI commands to sign various things. CLI commands are not
a direct part of the node, but rather a part of the epicchainGo binary; their
implementations use RPC to query data from the blockchain and perform any
required actions.