#!/usr/bin/env bash

docker --version
docker-compose --version

function check() {
  if [[ $1 -ne 0 ]] ; then
      exit 1
  fi
}

function caduceator-docker {
    CADUCEATOR_VERSION="$(make version -s)"
    make docker
    check $?
}

function deploy {
    echo "Deploying Cluster"
    git clone https://github.com/xmidt-org/codex-deploy.git 2> /dev/null || true
    pushd codex-deploy/deploy/docker-compose
    CADUCEATOR_VERSION=$CADUCEATOR_VERSION docker-compose up -d yb-master yb-tserver db-init caduceator
    check $?
    popd
    printf "\n"
}

caduceator-docker
cd ..

echo "Caduceator V:$CADUCEATOR_VERSION"
deploy
go get -d github.com/xmidt-org/codex-deploy/tests/...
printf "Starting Tests \n\n\n"
go run github.com/xmidt-org/codex-deploy/tests/runners/travis -feature=codex-deploy/tests/features/caduceator/travis
check $?
