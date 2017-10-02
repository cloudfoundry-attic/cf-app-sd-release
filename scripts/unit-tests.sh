#!/bin/bash

set -e -u -x

cd service-discovery-release
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin

go get github.com/onsi/ginkgo/ginkgo

ginkgo -r -race -randomizeAllSpecs -randomizeSuites src/bosh-dns-adapter