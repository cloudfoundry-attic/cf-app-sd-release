#!/usr/bin/env bash

set -eu

export GOPATH=$PWD/service-discovery-release
export PATH="${GOPATH}/bin":$PATH
export APPS_DIR=$PWD/service-discovery-release/src/example-apps

go install github.com/onsi/ginkgo/ginkgo

# replace admin password and secret in test config
VARS_STORE=${PWD}/vars-store/environments/${ENVIRONMENT_NAME}/vars-store.yml
pushd test-config/environments/${ENVIRONMENT_NAME}
  ADMIN_PASSWORD=`grep cf_admin_password ${VARS_STORE} | cut -d' ' -f2`
  sed -i -- "s/{{admin-password}}/${ADMIN_PASSWORD}/g" test-config.json
  ADMIN_SECRET=`grep uaa_admin_client_secret ${VARS_STORE} | cut -d' ' -f2`
  sed -i -- "s/{{admin-secret}}/${ADMIN_SECRET}/g" test-config.json
popd

ENVIRONMENT_PATH="test-config/environments/${ENVIRONMENT_NAME}/test-config.json"
export CONFIG=${PWD}/${CONFIG:-"${ENVIRONMENT_PATH}"}


pushd $GOPATH/src/acceptance
    ginkgo -keepGoing -randomizeAllSpecs -randomizeSuites -race .
popd
