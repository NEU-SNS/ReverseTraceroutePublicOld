#!/bin/bash

cd ../../cmd/vpservice

go build -a || exit 1

cp vpservice ./docker
cp vpservice.config ./docker
cp ./certs/* ./docker
cd docker

docker build --rm=true -t revtr/vpservice .
docker save -o vps.tar revtr/vpservice
docker rmi revtr/vpservice

rm vpservice
rm vpserv.key
rm vpserv.crt
rm root.crt
rm vpservice.config
