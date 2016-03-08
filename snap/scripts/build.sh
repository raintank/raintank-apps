#!/bin/sh

SNAPDIR=$GOPATH/src/github.com/intelsdi-x/snap

$SNAPDIR/scripts/build-plugin.sh $SNAPDIR/build/plugin github.com/raintank/raintank-apps/snap/plugin/snap-publisher-rt-hostedtsdb

$SNAPDIR/scripts/build-plugin.sh $SNAPDIR/build/plugin github.com/raintank/raintank-apps/snap/plugin/snap-collector-rt-gitstats

$SNAPDIR/scripts/build-plugin.sh $SNAPDIR/build/plugin github.com/raintank/raintank-apps/snap/plugin/snap-collector-worldping-ping