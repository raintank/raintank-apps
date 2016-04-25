#!/bin/sh
set -x
BASE=$(dirname $0)
CODE_DIR=${1:-$(readlink -e "$BASE/../../")}
SNAPDIR=$GOPATH/src/github.com/intelsdi-x/snap
BINDIR=${CODE_DIR}/build/plugins
mkdir -p $BINDIR
$SNAPDIR/scripts/build-plugin.sh $BINDIR github.com/raintank/raintank-apps/snap/plugin/snap-publisher-rt-hostedtsdb

$SNAPDIR/scripts/build-plugin.sh $BINDIR github.com/raintank/raintank-apps/snap/plugin/snap-collector-rt-gitstats

$SNAPDIR/scripts/build-plugin.sh $BINDIR github.com/raintank/raintank-apps/snap/plugin/snap-collector-worldping-ping

$SNAPDIR/scripts/build-plugin.sh $BINDIR github.com/raintank/raintank-apps/snap/plugin/snap-collector-ns1