#!/bin/bash
set -x

PKG=${1:-"task-server task-agent-ng"}

BASE=$(dirname $0)

# Detect OS, use readlink/greadlink
platform='linux'
unamestr=`uname`
if [[ "$unamestr" == 'Darwin' ]]; then
   platform='Darwin'
fi

READLINK="readlink"
if [[ $platform == 'Darwin' ]]; then
   READLINK="greadlink"
fi

CODE_DIR=$($READLINK -e "$BASE/../")

CURRENT_PWD=$(pwd)
cd $CODE_DIR

GIT_HASH=$(git rev-parse HEAD)

mkdir -p ${CODE_DIR}/build/bin/

for VAR in $PKG; do
	cd $CODE_DIR/$VAR
	go build -ldflags "-X main.GitHash=$GIT_HASH" -o ${CODE_DIR}/build/bin/$VAR
done

cd $CURRENT_PWD
