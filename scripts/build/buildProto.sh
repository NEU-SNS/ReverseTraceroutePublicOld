#!/bin/bash

GOSRC=$GOPATH/src

protoc  -I $GOSRC $GOSRC/github.com/NEU-SNS/ReverseTraceroute/datamodel/*.proto \
    --go_out=plugins=grpc:$GOSRC
protoc  -I $GOSRC $GOSRC/github.com/NEU-SNS/ReverseTraceroute/controllerapi/*.proto \
    --go_out=plugins=grpc:$GOSRC
protoc  -I $GOSRC $GOSRC/github.com/NEU-SNS/ReverseTraceroute/plcontrollerapi/*.proto \
    --go_out=plugins=grpc:$GOSRC
protoc  -I $GOSRC $GOSRC/github.com/NEU-SNS/ReverseTraceroute/vpservice/pb/*.proto \
    --go_out=plugins=grpc:$GOSRC
protoc  -I $GOSRC $GOSRC/github.com/NEU-SNS/ReverseTraceroute/atlas/pb/*.proto \
    --go_out=plugins=grpc:$GOSRC
