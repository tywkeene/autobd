#!/bin/bash

build_image(){
    echo "Building $1..."
    bash ./scripts/build/build.sh
    docker build --rm -t autobd:$1 -f docker/Dockerfile.$1 .
}

build_image $1
