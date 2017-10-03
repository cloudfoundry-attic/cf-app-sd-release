#!/bin/bash

set -eux

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd $DIR/..
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin

go get github.com/onsi/ginkgo/ginkgo

ginkgo -r -race -randomizeAllSpecs -randomizeSuites src/bosh-dns-adapter src/service-discovery-controller