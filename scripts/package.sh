#!/bin/bash

BASE=$(dirname $0)
CODE_DIR=$(readlink -e "$BASE/../")

BUILD_ROOT=$CODE_DIR/build

ARCH="$(uname -m)"
VERSION=$(git describe --long)

# ubuntu 14.04
for VAR in task-server task-agent; do
	BUILD="$BUILD_ROOT/upstart"
	NSQ_BUILD="${BUILD}/$VAR-${VERSION}"
	NSQ_PACKAGE_NAME="${BUILD}/${VAR}-${VERSION}_${ARCH}.deb"
	mkdir -p ${NSQ_BUILD}/usr/bin
	mkdir -p ${NSQ_BUILD}/etc/init
	mkdir -p ${NSQ_BUILD}/etc/raintank

	if [ $VAR == 'task-agent' ]; then
		# also add the plugins
		mkdir -p ${NSQ_BUILD}/var/lib/snap/plugins
		cp ${BUILD_ROOT}/plugins/* ${NSQ_BUILD}/var/lib/snap/plugins/
	fi

	cp ${BASE}/etc/${VAR}.ini ${NSQ_BUILD}/etc/raintank/
	cp ${BUILD_ROOT}/bin/$VAR ${NSQ_BUILD}/usr/bin
	fpm -s dir -t deb \
	  -v ${VERSION} -n ${VAR} -a ${ARCH} --description "Raintank $VAR" \
	  --deb-upstart ${BASE}/etc/init/${VAR} \
	  --config-files /etc/raintank/ \
	  -m "Raintank Inc. <hello@raintank.io>" --vendor "raintank.io" \
	  -C ${NSQ_BUILD} -p ${NSQ_PACKAGE_NAME} .
done

# ubuntu 16.04
for VAR in task-server task-agent; do
	BUILD="$BUILD_ROOT/systemd"
	NSQ_BUILD="${BUILD}/$VAR-${VERSION}"
	NSQ_PACKAGE_NAME="${BUILD}/${VAR}-${VERSION}_${ARCH}.deb"
	mkdir -p ${NSQ_BUILD}/usr/bin
	mkdir -p ${NSQ_BUILD}/lib/systemd/system
	mkdir -p ${NSQ_BUILD}/etc/raintank

	if [ $VAR == 'task-agent' ]; then
		# also add the plugins
		mkdir -p ${NSQ_BUILD}/var/lib/snap/plugins
		cp ${BUILD_ROOT}/plugins/* ${NSQ_BUILD}/var/lib/snap/plugins/
	fi

	cp ${BASE}/etc/${VAR}.ini ${NSQ_BUILD}/etc/raintank/
	cp ${BASE}/lib/systemd/system/${VAR}.service ${NSQ_BUILD}/lib/systemd/system
	cp ${BUILD_ROOT}/bin/$VAR ${NSQ_BUILD}/usr/bin
	fpm -s dir -t deb \
	  -v ${VERSION} -n ${VAR} -a ${ARCH} --description "Raintank $VAR" \
	  --config-files /etc/raintank/ \
	  -m "Raintank Inc. <hello@raintank.io>" --vendor "raintank.io" \
	  -C ${NSQ_BUILD} -p ${NSQ_PACKAGE_NAME} .
done
