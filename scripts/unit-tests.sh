#!/bin/bash

set -eux

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd $DIR/..
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin

go get github.com/onsi/ginkgo/ginkgo

echo -e "\n Formatting packages..."

for packageToFmt in bosh-dns-adapter service-discovery-controller acceptance_tests; do
    reformatted_packages=$(go fmt $packageToFmt/...)
    if [[ $reformatted_packages = *[![:space:]]* ]]; then
      echo "FAILURE: go fmt reformatted the following packages:"
      echo $reformatted_packages
      exit 1
    fi
done

ginkgo -r -race -randomizeAllSpecs -randomizeSuites src/bosh-dns-adapter src/service-discovery-controller