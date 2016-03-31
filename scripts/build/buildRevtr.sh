#!/bin/bash

cd ../../cmd/revtr

go build -a || exit 1

cp revtr ./docker
cp revtr.config ./docker
cp ./certs/* ./docker
cp -r ./webroot ./docker
cd docker

docker build --rm=true -t revtr/revtr .
docker save -o revtr.tar revtr/revtr
docker rmi revtr/revtr

rm revtr
rm root.crt
rm revtr.crt
rm revtr.key
rm revtr.config
rm -rf webroot
