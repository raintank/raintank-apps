#!/bin/sh
set -x
BASE=$(dirname $0)
CODE_DIR=${1:-$(readlink -e "$BASE/../../")}
SNAPDIR=$GOPATH/src/github.com/intelsdi-x/snap
BINDIR=${CODE_DIR}/build/plugins
mkdir -p $BINDIR

cd $CODE_DIR/snap/plugin
for p in *; do
	cd $CODE_DIR/snap/plugin/$p
	go build -a -ldflags "-w" -o $BINDIR/$p
done
