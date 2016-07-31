#!/bin/bash

# Find the directory we exist within
BASE=$(dirname $0)
CODE_DIR=$(readlink -e "$BASE/../")

if [ -z ${PACKAGECLOUD_REPO} ] ; then
  echo "The environment variable PACKAGECLOUD_REPO must be set."
  exit 1
fi

package_cloud push ${PACKAGECLOUD_REPO}/ubuntu/trusty ${CODE_DIR}/build/upstart/*.deb
package_cloud push ${PACKAGECLOUD_REPO}/ubuntu/xenial ${CODE_DIR}/build/systemd/*.deb
