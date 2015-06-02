#!/bin/bash

ROOT=`pwd`

CONT=../src/controller
PLCONT=../src/plcontroller
PLVP=../src/plvp
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
