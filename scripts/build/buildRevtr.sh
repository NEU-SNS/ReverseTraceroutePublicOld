#!/bin/bash

cd ../../cmd/revtr

go build -a || exit 1

cp revtr ./docker
cp revtr.config ./docker
cp ./certs/* ./docker
cp -r ./webroot ./docker
cd docker

docker build --rm=true -t revtr/new .
docker save -o newrtr.tar revtr/new
docker rmi revtr/new

rm revtr
rm root.crt
rm revtr.crt
rm revtr.key
rm revtr.config
rm -rf webroot
