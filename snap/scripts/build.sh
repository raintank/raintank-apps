#!/bin/sh
BASE=$(dirname $0)
BINDIR=$(readlink -e "$BASE/../bin")
SNAPDIR=$GOPATH/src/github.com/intelsdi-x/snap

mkdir -p $BINDIR
$SNAPDIR/scripts/build-plugin.sh $BINDIR github.com/raintank/raintank-apps/snap/plugin/snap-publisher-rt-hostedtsdb

$SNAPDIR/scripts/build-plugin.sh $BINDIR github.com/raintank/raintank-apps/snap/plugin/snap-collector-rt-gitstats

$SNAPDIR/scripts/build-plugin.sh $BINDIR github.com/raintank/raintank-apps/snap/plugin/snap-collector-worldping-ping
