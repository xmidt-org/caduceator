## SPDX-FileCopyrightText: 2020 Comcast Cable Communications Management, LLC
## SPDX-License-Identifier: Apache-2.0
#!/bin/bash

DIR=$( cd $(dirname $0) ; pwd -P )
ROOT_DIR=$DIR/../../

echo "Running services..."
CADUCEUS_VERSION=${CADUCEUS_VERSION:-latest} \
ARGUS_VERSION=${ARGUS_VERSION:-latest} \
CADUCEATOR_VERSION=${CADUCEATOR_VERSION:-latest} \
docker-compose -f $ROOT_DIR/deploy/docker-compose/docker-compose.yml up -d $@

sleep 10

bash config_dynamodb.sh

