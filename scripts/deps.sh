#!/bin/bash

BASE=$(readlink -e $(dirname $0))

CIRCLE_PROJECT_USERNAME=${CIRCLE_PROJECT_USERNAME:-raintank}
CIRCLE_PROJECT_REPONAME=${CIRCLE_PROJECT_REPONAME:-raintank-apps}

mkdir -p /home/ubuntu/go/src/github.com/$CIRCLE_PROJECT_USERNAME
ln -s /home/ubuntu/$CIRCLE_PROJECT_REPONAME /home/ubuntu/go/src/github.com/$CIRCLE_PROJECT_USERNAME/$CIRCLE_PROJECT_REPONAME

curl https://glide.sh/get | sh

mkdir -p /home/ubuntu/go/src/github.com/intelsdi-x/
cd /home/ubuntu/go/src/github.com/intelsdi-x/
git clone https://github.com/intelsdi-x/snap.git
cd /home/ubuntu/go/src/github.com/intelsdi-x/snap
git checkout ca32c9af5b93d79f1b559469cc163258b1989b2d
make deps

go get github.com/intelsdi-x/snap-plugin-lib-go/...
cd /home/ubuntu/go/src/github.com/intelsdi-x/snap-plugin-lib-go
glide up

cd /home/ubuntu/go/src/github.com/$CIRCLE_PROJECT_USERNAME/$CIRCLE_PROJECT_REPONAME
go get -t ./...


cd $BASE
bundle install
