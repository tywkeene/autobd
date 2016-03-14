#!/bin/bash

function rm_container(){
    if docker ps -f name='$1' &> /dev/null; then
        echo "Removing old $(docker rm -f $1)"
    fi
}

function build_image(){
    echo "Building $1..."
    docker build --rm -t autobd:$1 -f docker/Dockerfile.node .
}

if [ -z "$1" ]; then
    printf "Usage $0 <number of nodes to create>\n"
    exit -1
fi

build_image "autobd-node"

for i in `seq 1 $1`; do
    [ -d "/home/$USER/data/autobd-nodes/node$i" ] && rm -rf /home/$user/data/autobd-nodes/node$i
    mkdir /home/$USER/data/autobd-nodes/node$i
    docker run -d --net frontend -v /home/$USER/data/autobd-nodes/node$i:/home/autobd-node/data --name "autobd-node$i" autobd:node
done
