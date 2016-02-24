#!/bin/bash

cd ../../cmd/controller

go build -a

cp controller ./docker
cp controller.config ./docker
cp certs/controller.crt ./docker
cp certs/controller.key ./docker
cp certs/root.crt ./docker
cd docker

docker build --rm=true -t rhansen2/controller .

rm controller
rm controller.config
rm controller.crt
rm controller.key
rm root.crt
