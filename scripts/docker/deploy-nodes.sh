#!/usr/bin/env bash

if [ -z "$1" ]; then
    printf "Usage $0 <number of nodes to create>\n"
    exit -1
fi

bash ./scripts/docker/build_image.sh "node"

ETC_DIR="/home/$USER/etc/autobd"
DATA_DIR="/home/$USER/data/autobd-nodes/node"

for i in `seq 1 $1`; do
    if [ ! -d "$DATA_DIR$1" ]; then
        mkdir -p "$DATA_DIR$i" && echo "Created $DATA_DIR$i"
    fi
    echo "CREATE autobd-node$i: $(docker run -d \
        --network autobd \
        -v $DATA_DIR$i:/home/autobd/data \
        -v $ETC_DIR:/home/autobd/etc \
        --name "autobd-node$i" autobd:node)"
done
