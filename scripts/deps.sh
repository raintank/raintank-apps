#!/bin/bash
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

BASE=$($READLINK -e $(dirname $0))

CIRCLE_PROJECT_USERNAME=${CIRCLE_PROJECT_USERNAME:-raintank}
CIRCLE_PROJECT_REPONAME=${CIRCLE_PROJECT_REPONAME:-raintank-apps}

#mkdir -p $GOPATH/src/github.com/$CIRCLE_PROJECT_USERNAME
#ln -s $HOME/$CIRCLE_PROJECT_REPONAME $GOPATH/src/github.com/$CIRCLE_PROJECT_USERNAME/$CIRCLE_PROJECT_REPONAME

#curl https://glide.sh/get | sh

#mkdir -p $GOPATH/src/github.com/intelsdi-x/
#cd $GOPATH/src/github.com/intelsdi-x/
#git clone https://github.com/intelsdi-x/snap.git
#cd $GOPATH/src/github.com/intelsdi-x/snap
#git checkout 2439ea1b2b12d1f13b2df7b3cf1b85475feadf44
#make deps

#go get github.com/intelsdi-x/snap-plugin-lib-go/...
#cd $GOPATH/src/github.com/intelsdi-x/snap-plugin-lib-go
#glide up

#mkdir -p $GOPATH/src/github.com/google
#cd $GOPATH/src/github.com/google
#git clone https://github.com/google/go-github
#cd go-github
#git checkout e7bb4b8ce29fb7beaf0765acda602bc516a56dd5

cd $GOPATH/src/github.com/$CIRCLE_PROJECT_USERNAME/$CIRCLE_PROJECT_REPONAME
go get -t ./...

#go get github.com/go-xorm/xorm
cd $GOPATH/src/github.com/go-xorm/xorm
git checkout v0.5.6
#git checkout 9bf34c31890cb518c714bedfec324cdfaacc4cf7

cd $BASE
#bundle install
