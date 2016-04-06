#!/bin/bash

BASE=$(dirname $0)
CODE_DIR=$(readlink -e "$BASE/../")

BUILD=$CODE_DIR/build

VERSION="0.0.1" # need an automatic way to do this again :-/
ARCH="$(uname -m)"
ITERATION=`date +%s`ubuntu1
TAG="pkg-${VERSION}-${ITERATION}"

for VAR in task-server task-agent tsdb; do
	NSQ_BUILD="${BUILD}/$VAR-${VERSION}"
	NSQ_PACKAGE_NAME="${BUILD}/${VAR}-${VERSION}_${ITERATION}_${ARCH}.deb"
	mkdir -p ${NSQ_BUILD}/usr/bin
	mkdir -p ${NSQ_BUILD}/etc/init
	mkdir -p ${NSQ_BUILD}/etc/raintank

	cp ${BASE}/etc/${VAR}.ini ${NSQ_BUILD}/etc/raintank/
	cp ${BUILD}/bin/$VAR ${NSQ_BUILD}/usr/bin
	fpm -s dir -t deb \
	  -v ${VERSION} -n ${VAR} -a ${ARCH} --iteration $ITERATION --description "Raintank $VAR" \
	  --deb-upstart ${BUILD}/etc/init/${VAR} \
	  -C ${NSQ_BUILD} -p ${NSQ_PACKAGE_NAME} .
done

git tag $TAG