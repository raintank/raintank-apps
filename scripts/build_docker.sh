#!/bin/bash

set -x
# Find the directory we exist within
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

VERSION=`git describe --abbrev=7`

# regular image
rm -rf build/*
mkdir -p build
cp ../build/* build/


docker build -t raintank/raintank-apps-task-server -f docker/Dockerfile-task-server .
docker tag raintank/raintank-apps-task-server raintank/raintank-apps-task-server:latest
docker tag raintank/raintank-apps-task-server raintank/raintank-apps-task-server:$VERSION

docker build -t raintank/raintank-apps-task-agent-ng -f docker/Dockerfile-task-agent-ng .
docker tag raintank/raintank-apps-task-agent-ng raintank/raintank-apps-task-agent-ng:latest
docker tag raintank/raintank-apps-task-agent-ng raintank/raintank-apps-task-agent-ng:$VERSION
