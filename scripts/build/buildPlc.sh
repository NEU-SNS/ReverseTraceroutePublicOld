#!/bin/bash

cd ../../cmd/plcontroller

go build

cp plcontroller ./docker
cp plcontroller.config ./docker
cp ./certs/* ./docker
cd docker
cp ../../../id_rsa_pl .

docker build --rm=true -t rhansen2/plcontroller .

rm id_rsa_pl
rm plcontroller
rm plcontroller.config
rm plcontroller.crt
rm plcontroller.key
