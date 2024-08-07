# EpicChain Go Node and SDK

<p align="center">
  <b>EpicChain</b> Node and SDK for the <a href="https://epicchain.org">EpicChain</a> blockchain.
</p>

<hr />

[![codecov](https://codecov.io/gh/epicchainlabs/epicchain-go/branch/master/graph/badge.svg)](https://codecov.io/gh/epicchainlabs/epicchain-go)
[![GithubWorkflows Tests](https://github.com/epicchainlabs/epicchain-go/actions/workflows/tests.yml/badge.svg)](https://github.com/epicchainlabs/epicchain-go/actions/workflows/tests.yml)
[![Report](https://goreportcard.com/badge/github.com/epicchainlabs/epicchain-go)](https://goreportcard.com/report/github.com/epicchainlabs/epicchain-go)
[![GoDoc](https://godoc.org/github.com/epicchainlabs/epicchain-go?status.svg)](https://godoc.org/github.com/epicchainlabs/epicchain-go)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/epicchainlabs/epicchain-go?sort=semver)
![License](https://img.shields.io/github/license/epicchainlabs/epicchain-go.svg?style=popout)

## Overview

Welcome to **EpicChainGo**, the comprehensive platform for developing distributed applications on the EpicChain blockchain. EpicChainGo provides an extensive suite of tools and components essential for blockchain development, including but not limited to:

- **Consensus Node**: Handle the consensus mechanism of the EpicChain network. For detailed information, check out the [consensus documentation](docs/consensus.md).
- **RPC Node & Client**: Facilitate remote procedure calls to interact with the EpicChain blockchain. For more details, refer to [RPC documentation](docs/rpc.md).
- **CLI Tool**: Command-line interface tool for various blockchain operations. Instructions can be found in the [CLI documentation](docs/cli.md).
- **Smart Contract Compiler**: Compile smart contracts written in Go into EpicChain VM bytecode. For guidance, see [compiler documentation](docs/compiler.md).
- **EpicChain Virtual Machine**: The virtual machine that executes smart contracts. More information is available in the [VM documentation](docs/vm.md).
- **Smart Contract Examples**: Explore various smart contract examples to help you get started. See the [examples directory](examples/README.md) for more.
- **Oracle Service**: Service for providing external data to the blockchain. Learn more in the [oracle documentation](docs/oracle.md).
- **State Validation Service**: Ensure the validity of the blockchain state. Detailed information can be found in the [state validation documentation](docs/stateroots.md).

EpicChainGo is designed to be compatible with the EpicChain N3 protocol. However, if you need to work with the EpicChain Legacy protocol, you can find an implementation in the [**master-2.x** branch](https://github.com/epicchainlabs/epicchain-go/tree/master-2.x) and releases prior to version 0.80.0 (version **0.7X.Y** track).

## Getting Started

### Installation

EpicChainGo is distributed as a single binary, encompassing all the features and functionality you need (please note that the smart contract compiler requires the Go compiler). You have several options for installation:

1. **Download the Binary**: Obtain the pre-built binary from the [releases page](https://github.com/epicchainlabs/epicchain-go/releases).
2. **Docker Image**: Use the Docker image available on [Docker Hub](https://hub.docker.com/r/nspccdev/epicchain-go). The `:latest` tag points to the most recent release.
3. **Build from Source**: Compile the binary yourself by following the instructions below.

#### Building from Source

To build EpicChainGo from source, you need Go 1.20+ and `make` installed:

```sh
make
```

The build process will generate the binary located at `bin/epicchain-go`. We recommend using tagged releases rather than random revisions from the `master` branch, as these may contain bugs or incompatibilities depending on the development stage.

##### Building on Windows

For building EpicChainGo on Windows, install `make` from the [MinGW package](https://osdn.net/projects/mingw/). Then, you can build EpicChainGo with:

```sh
make
```

The resulting binary will be `bin/epicchain-go.exe`.

### Running a Node

To run an EpicChain node, it must be connected to a network, either a local network (commonly referred to as `privnet`) or a public network (such as `mainnet` or `testnet`). Network configurations are managed via files, and EpicChainGo supports storing multiple configuration files in one directory (`./config` by default) and switching between them using network flags.

To start a node on a private network, use:

```sh
./bin/epicchain-go node
```

To specify a different network, use an appropriate flag:

```sh
./bin/epicchain-go node --mainnet
```

Available network flags include:
- `--mainnet, -m`
- `--privnet, -p`
- `--testnet, -t`

For running a consensus/committee node, please refer to the [consensus documentation](docs/consensus.md).

If you're using Windows, ensure that Windows Firewall is configured to allow inbound connections to the P2P port.

### Docker

By default, the Docker image is configured to run a node on `privnet`. To start a node using Docker, execute:

```bash
docker run -d --name epicchain-go -p 20332:20332 -p 20331:20331 nspccdev/epicchain-go
```

This command starts a node on `privnet` and exposes ports `20332` (for P2P protocol) and `20331` (for JSON-RPC server).

### Importing Chain Dumps

To initialize your mainnet or testnet node with [chain archives provided by NGD](https://sync.ngd.network/), follow these steps:

```sh
$ wget .../chain.acc.zip # download the chain dump file
$ unzip chain.acc.zip
$ ./bin/epicchain-go db restore -m -i chain.acc # use '-t' flag instead of '-m' for testnet
```

The import process differs from that of the C# node in that block importing is a separate mode. After importing, you can start the node normally.

## Running a Private Network

For detailed instructions on running a private network, refer to the [consensus node documentation](docs/consensus.md).

## Smart Contract Development

For guidance on developing smart contracts with EpicChainGo, visit the [EpicChainGo smart contract development workshop](https://github.com/epicchainlabs/epicchain-go-sc-wrkshp). This workshop provides examples of simple contracts that can be compiled, deployed, and run using the EpicChainGo compiler, SDK, and a private network. For specifics on translating Go code to EpicChain VM bytecode and contract limitations, consult the [compiler documentation](docs/compiler.md).

Explore more EpicChain smart contract examples written in Go in the [examples directory](examples/README.md).

