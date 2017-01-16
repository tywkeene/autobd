#!/bin/bash

build_image(){
    echo "Building $1..."
    bash ./build.sh
    docker rmi -f autobd:$1
    docker build --rm -t autobd:$1 -f docker/Dockerfile.$1 .
}

build_image $1
