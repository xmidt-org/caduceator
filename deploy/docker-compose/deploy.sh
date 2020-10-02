#!/bin/bash

DIR=$( cd $(dirname $0) ; pwd -P )
ROOT_DIR=$DIR/../../

pushd $ROOT_DIR
echo "Building Simulator..."
docker build -t simulator:local $ROOT_DIR/simulator
popd

echo "Running services..."
CADUCEUS_VERSION=${CADUCEUS_VERSION:-local} \
ARGUS_VERSION=${ARGUS_VERSION:-0.3.6} \
CADUCEATOR_VERSION=${CADUCEATOR_VERSION:-local} \
SIMULATOR_VERSION=${SIMULATOR_VERSION:-local} \
docker-compose -f $ROOT_DIR/deploy/docker-compose/docker-compose.yml up -d $@

sleep 10

bash config_dynamodb.sh


# TR1D1UM_VERSION=${TR1D1UM_VERSION:-0.1.5} \
# CADUCEUS_VERSION=${CADUCEUS_VERSION:-0.2.1} \

# echo "Building goaws..."
# git clone git@github.com:kcajmagic/goaws.git /tmp/goaws
# pushd /tmp/goaws
# git checkout adding_http_support
# docker build -t goaws:local .
# popd
