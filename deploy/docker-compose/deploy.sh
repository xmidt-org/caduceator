#!/bin/bash

DIR=$( cd $(dirname $0) ; pwd -P )
ROOT_DIR=$DIR/../../

echo "Running services..."
CADUCEUS_VERSION=${CADUCEUS_VERSION:-local} \
ARGUS_VERSION=${ARGUS_VERSION:-0.3.6} \
CADUCEATOR_VERSION=${CADUCEATOR_VERSION:-local} \
docker-compose -f $ROOT_DIR/deploy/docker-compose/docker-compose.yml up -d $@

sleep 10

bash config_dynamodb.sh

