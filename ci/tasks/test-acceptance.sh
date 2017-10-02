#!/usr/bin/env bash

set -exu

ROOT_DIR=$PWD

start-bosh -o /usr/local/bosh-deployment/local-dns.yml

source /tmp/local-bosh/director/env

bosh int /tmp/local-bosh/director/creds.yml --path /jumpbox_ssh/private_key > /tmp/jumpbox_ssh_key.pem
chmod 400 /tmp/jumpbox_ssh_key.pem

export BOSH_GW_PRIVATE_KEY="/tmp/jumpbox_ssh_key.pem"
export BOSH_GW_USER="jumpbox"
export BOSH_DIRECTOR_IP="10.245.0.3"
export BOSH_BINARY_PATH=$(which bosh)
export BOSH_DEPLOYMENT="acceptance"
export TEST_CLOUD_CONFIG_PATH="/tmp/cloud-config.yml"

bosh -n update-cloud-config /usr/local/bosh-deployment/docker/cloud-config.yml -v network=director_network

bosh upload-stemcell https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-trusty-go_agent

pushd $ROOT_DIR/service-discovery-release
   bosh create-release --force
   bosh upload-release
popd

pushd $ROOT_DIR/bosh-dns-release
   bosh create-release --force
   bosh upload-release
popd

export GOPATH=$PWD/service-discovery-release
export PATH="${GOPATH}/bin":$PATH

go install github.com/onsi/ginkgo/ginkgo

pushd $GOPATH/src/acceptance_tests
    ginkgo -keepGoing -randomizeAllSpecs -randomizeSuites -race .
popd
