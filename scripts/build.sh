#!/bin/bash
set -x

PKG=${1:-"task-server task-agent"}

BASE=$(dirname $0)

CODE_DIR=$(readlink -e "$BASE/../")

CURRENT_PWD=$(pwd)
cd $CODE_DIR

GIT_HASH=$(git rev-parse HEAD)

mkdir -p ${CODE_DIR}/build/bin/

for VAR in $PKG; do
	cd $CODE_DIR/$VAR
	go build -ldflags "-X main.GitHash=$GIT_HASH" -o ${CODE_DIR}/build/bin/$VAR
done

cd $CURRENT_PWD
