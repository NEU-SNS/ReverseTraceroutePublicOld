#!/bin/bash

cd ../../cmd/atlas

go build -a || exit 1

cp atlas ./docker
cp atlas.config ./docker
cp ./certs/* ./docker
cd docker

docker build --rm=true -t revtr/atlas .
docker save -o atlas.tar revtr/atlas
docker rmi revtr/atlas

rm atlas
rm atlas.config
rm atlas.crt
rm atlas.key
rm root.crt
