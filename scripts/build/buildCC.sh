#!/bin/bash

cd ../../cmd/controller

go build

cp controller ./docker
cp controller.config ./docker
cd docker

docker build --rm=true -t rhansen2/controller .

rm controller
rm controller.config
