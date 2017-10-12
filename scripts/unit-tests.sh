#!/bin/bash

set -eux

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd $DIR/..
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin

go get github.com/onsi/ginkgo/ginkgo

ignored=(vendor,Tools,bin,ci,docs,gobin,out,test,tmp)
echo -e "\n Formatting packages, other than: ${ignored[*]}..."
for i in `ls -1` ; do
  if [ -d "$i" ] && [[ ! ${ignored[*]} =~ "$i" ]] ; then
    go fmt github.com/cloudfoundry/bosh-agent/${i}/...
  fi
done

ginkgo -r -race -randomizeAllSpecs -randomizeSuites src/bosh-dns-adapter src/service-discovery-controller