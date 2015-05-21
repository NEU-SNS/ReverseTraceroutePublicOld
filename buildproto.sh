#!/bin/bash

GOSRC=$GOPATH/src

protoc  -I $GOSRC $GOSRC/github.com/NEU-SNS/ReverseTraceroute/lib/datamodel/*.proto \
    --go_out=plugins=grpc:$GOSRC
protoc  -I $GOSRC $GOSRC/github.com/NEU-SNS/ReverseTraceroute/lib/controller/*.proto \
    --go_out=plugins=grpc:$GOSRC

