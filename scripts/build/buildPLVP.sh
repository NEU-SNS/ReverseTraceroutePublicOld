#!/bin/bash

cd ../../cmd/plvp

GOOS=linux GOARCH=386 go build -a -ldflags \
    "-X main.versionNo=$1 -X main.pidFile=$2 -X main.lockFile=$3 -X main.configPath=/home/uw_geoloc4/plvp/plvp.config"
