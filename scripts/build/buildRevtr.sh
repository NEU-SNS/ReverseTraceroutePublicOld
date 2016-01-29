#!/bin/bash

cd ../../cmd/revtr

go build

cp revtr ./docker
cp revtr.config ./docker
cp ./certs/* ./docker
cp -r ./webroot ./docker
cd docker

docker build --rm=true -t rhansen2/revtr .

rm revtr
rm root.crt
rm revtr.config
rm -rf webroot
