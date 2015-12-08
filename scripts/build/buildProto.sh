#!/bin/bash

GOSRC=$GOPATH/src

protoc  -I $GOSRC $GOSRC/github.com/NEU-SNS/ReverseTraceroute/datamodel/*.proto \
    --go_out=plugins=grpc:$GOSRC
protoc  -I $GOSRC $GOSRC/github.com/NEU-SNS/ReverseTraceroute/controller/pb/*.proto \
    --go_out=plugins=grpc:$GOSRC
protoc  -I $GOSRC $GOSRC/github.com/NEU-SNS/ReverseTraceroute/plcontroller/pb/*.proto \
    --go_out=plugins=grpc:$GOSRC
protoc  -I $GOSRC $GOSRC/github.com/NEU-SNS/ReverseTraceroute/vpservice/pb/*.proto \
    --go_out=plugins=grpc:$GOSRC
protoc  -I $GOSRC $GOSRC/github.com/NEU-SNS/ReverseTraceroute/atlas/pb/*.proto \
    --go_out=plugins=grpc:$GOSRC
