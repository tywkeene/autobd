#!/bin/bash

function build_image(){
    echo "Building $1..."
    docker build --rm -t autobd:$1 -f docker/Dockerfile.node .
}

if [ -z "$1" ]; then
    printf "Usage $0 <number of nodes to create>\n"
    exit -1
fi

build_image "node"

SEED_SERVER="https://172.18.0.2:8080"
for i in `seq 1 $1`; do
    if [ -d "/home/$USER/data/autobd-nodes/node$i" ]; then
        echo "removing /home/$USER/data/autobd-node/node$i"
        rm -rf /home/$USER/data/autobd-nodes/node$i
    fi
    mkdir /home/$USER/data/autobd-nodes/node$i
    docker run -d --net autobd \
        -e "SEED_SERVER=$SEED_SERVER" \
        -v /home/$USER/data/autobd-nodes/node$i:/home/autobd-node/data \
        --name "autobd-node$i" autobd:node
done
