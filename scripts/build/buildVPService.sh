#!/bin/bash

cd ../../cmd/vpservice

go build

cp vpservice ./docker
cd docker

docker build --rm=true -t rhansen2/vpservice .

rm vpservice
