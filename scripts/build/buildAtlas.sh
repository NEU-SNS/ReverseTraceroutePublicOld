#!/bin/bash

cd ../../cmd/atlas

go build

cp atlas ./docker
cp atlas.config ./docker
cd docker

docker build --rm=true -t rhansen2/atlas .

rm atlas
rm atlas.config
