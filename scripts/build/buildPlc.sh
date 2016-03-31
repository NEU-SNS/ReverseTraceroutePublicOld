#!/bin/bash

cd ../../cmd/plcontroller

go build -a || exit 1

cp plcontroller ./docker
cp plcontroller.config ./docker
cp ./certs/* ./docker
cd docker
cp ../../../id_rsa_pl .

docker build --rm=true -t revtr/plcontroller .
docker save -o plc.tar revtr/plcontroller
docker rmi revtr/plcontroller

rm id_rsa_pl
rm plcontroller
rm plcontroller.config
rm plcontroller.crt
rm plcontroller.key
