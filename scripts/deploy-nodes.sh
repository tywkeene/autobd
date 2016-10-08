#!/bin/bash

build_image(){
    echo "Building $1..."
    docker build --rm -t autobd:$1 -f docker/Dockerfile.node .
}

if [ -z "$1" ]; then
    printf "Usage $0 <number of nodes to create>\n"
    exit -1
fi

build_image "node"

ETC_DIR="/home/$USER/etc/autobd"
DATA_DIR="/home/$USER/data/autobd-nodes/node"

for i in `seq 1 $1`; do
    if [ ! -d "$DATA_DIR$1" ]; then
        mkdir "$DATA_DIR$i"
        echo "$DATA_DIR$i"
    fi
    docker run -d \
        --network autobd \
        -v $DATA_DIR$i:/home/autobd/data \
        -v $ETC_DIR:/home/autobd/etc \
        --name "autobd-node$i" autobd:node
done
