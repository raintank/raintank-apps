#!/bin/bash

BASE=$(dirname $0)
CODE_DIR=$(readlink -e "$BASE/../")

BUILD=$CODE_DIR/build

ARCH="$(uname -m)"
VERSION=$(git describe --long)

for VAR in task-server task-agent; do
	NSQ_BUILD="${BUILD}/$VAR-${VERSION}"
	NSQ_PACKAGE_NAME="${BUILD}/${VAR}-${VERSION}_${ARCH}.deb"
	mkdir -p ${NSQ_BUILD}/usr/bin
	mkdir -p ${NSQ_BUILD}/etc/init
	mkdir -p ${NSQ_BUILD}/etc/raintank

	if [ $VAR == 'task-agent' ]; then
		# also add the plugins
		mkdir -p ${NSQ_BUILD}/var/lib/snap/plugins
		cp ${BUILD}/plugins/* ${NSQ_BUILD}/var/lib/snap/plugins/
	fi

	cp ${BASE}/etc/${VAR}.ini ${NSQ_BUILD}/etc/raintank/
	cp cp ${BUILD}/bin/$VAR ${NSQ_BUILD}/usr/bin
	fpm -s dir -t deb \
	  -v ${VERSION} -n ${VAR} -a ${ARCH} --description "Raintank $VAR" \
	  --deb-upstart ${BASE}/etc/init/${VAR} \
	  -C ${NSQ_BUILD} -p ${NSQ_PACKAGE_NAME} .
done
