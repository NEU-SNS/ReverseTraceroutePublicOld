#!/bin/bash

cd ../../cmd/atlas

go build

cp atlas ./docker
cp atlas.config ./docker
cp ./certs/* ./docker
cd docker

docker build --rm=true -t rhansen2/atlas .

rm atlas
rm atlas.config
rm atlas.crt
rm atlas.key
rm root.crt
