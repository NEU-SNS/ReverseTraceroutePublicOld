#!/bin/bash

cd ../../cmd/vpservice

go build

cp vpservice ./docker
cp vpservice.config ./docker
cp ./certs/* ./docker
cd docker

docker build --rm=true -t rhansen2/vpservice .

rm vpservice
rm vpserv.key
rm vpserv.crt
rm root.crt
rm vpservice.config
