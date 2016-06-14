#!/bin/bash

# build and save the docker image for the atlas

set -e 

cp $BIN_DIR/$BIN cmd/$BIN/docker/.
cd cmd/$BIN/docker
CONT_DIR=containers
ROOT=$(git rev-parse --show-toplevel)

docker build --rm=true -t revtr/atlas .
docker save -o $ROOT/$CONT_DIR/atlas.tar revtr/atlas
docker rmi revtr/atlas
