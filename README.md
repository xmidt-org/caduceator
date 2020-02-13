# Caduceator

[![Build Status](https://travis-ci.com/xmidt-org/caduceator.svg?branch=master)](https://travis-ci.com/xmidt-org/caduceator)
[![codecov.io](http://codecov.io/github/xmidt-org/caduceator/coverage.svg?branch=master)](http://codecov.io/github/xmidt-org/caduceator?branch=master)
[![Code Climate](https://codeclimate.com/github/xmidt-org/caduceator/badges/gpa.svg)](https://codeclimate.com/github/xmidt-org/caduceator)
[![Issue Count](https://codeclimate.com/github/xmidt-org/caduceator/badges/issue_count.svg)](https://codeclimate.com/github/xmidt-org/caduceator)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmidt-org/caduceator)](https://goreportcard.com/report/github.com/xmidt-org/caduceator)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/caduceator/blob/master/LICENSE)
[![GitHub release](https://img.shields.io/github/release/xmidt-org/caduceator.svg)](CHANGELOG.md)


## Summary

Caduceator provides a way to performance test [Caduceus](https://github.com/xmidt-org/caduceus),
which is a part of [XMiDT]((https://github.com/xmidt-org/xmidt)).

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Details](#details)
- [Build](#build)
- [Deploy](#deploy)
- [Contributing](#contributing)

## Code of Conduct

This project and everyone participating in it are governed by the [XMiDT Code Of Conduct](https://xmidt.io/code_of_conduct/). 
By participating, you agree to this Code.

## Details

TBD.

## Build

### Source

In order to build from the source, you need a working Go environment with 
version 1.13 or greater. Find more information on the [Go website](https://golang.org/doc/install).

You can directly use `go get` to put the Caduceator binary into your `GOPATH`:
```bash
go get github.com/xmidt-org/caduceator
```

You can also clone the repository yourself and build using make:

```bash
mkdir -p $GOPATH/src/github.com/xmidt-org
cd $GOPATH/src/github.com/xmidt-org
git clone git@github.com:xmidt-org/caduceator.git
cd caduceator
make build
```

### Makefile

The Makefile has the following options you may find helpful:
* `make build`: builds the Caduceator binary
* `make docker`: builds a docker image for Caduceator, making sure to get all 
   dependencies
* `make local-docker`: builds a docker image for Caduceator with the assumption
   that the dependencies can be found already
* `make it`: runs `make docker`, then deploys Caduceator and a cockroachdb 
   database into docker.
* `make test`: runs unit tests with coverage for Caduceator
* `make clean`: deletes previously-built binaries and object files

### RPM

First have a local clone of the source and go into the root directory of the 
repository.  Then use rpkg to build the rpm:
```bash
rpkg srpm --spec <repo location>/<spec file location in repo>
rpkg -C <repo location>/.config/rpkg.conf sources --outdir <repo location>'
```

### Docker

The docker image can be built either with the Makefile or by running a docker 
command.  Either option requires first getting the source code.

See [Makefile](#Makefile) on specifics of how to build the image that way.

For running a command, either you can run `docker build` after getting all 
dependencies, or make the command fetch the dependencies.  If you don't want to 
get the dependencies, run the following command:
```bash
docker build -t caduceator:local -f deploy/Dockerfile .
```
If you want to get the dependencies then build, run the following commands:
```bash
GO111MODULE=on go mod vendor
docker build -t caduceator:local -f deploy/Dockerfile.local .
```

For either command, if you want the tag to be a version instead of `local`, 
then replace `local` in the `docker build` command.

### Kubernetes

WIP. TODO: add info

## Deploy

For deploying on Docker or in Kubernetes, refer to the [deploy README](https://github.com/xmidt-org/codex-deploy/tree/master/deploy/README.md).

For running locally, ensure you have the binary [built](#Source). If the binary 
is in your current folder, run:
```
./caduceator
```

## Contributing

Refer to [CONTRIBUTING.md](CONTRIBUTING.md).
