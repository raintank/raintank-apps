#!/bin/bash

set -x
# Find the directory we exist within
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

VERSION=`git describe --abbrev=7`

docker push raintank/raintank-apps-task-server:$VERSION
docker push raintank/raintank-apps-task-server:latest

docker push raintank/raintank-apps-task-agent-ng:$VERSION
docker push raintank/raintank-apps-task-agent-ng:latest
