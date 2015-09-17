#!/bin/bash

ROOT=`pwd`

CONT=../cmd/controller
PLCONT=../cmd/plcontroller
PLVP=../cmd/plvp
BUILD="go build"
INSTALL="go install"
CLEAN="go clean"

TARGETS=($CONT $PLCONT $PLVP)

for TARGET in ${TARGETS[*]}
do
    cd $ROOT/$TARGET
    $BUILD && $INSTALL && $CLEAN
done

cd $ROOT
