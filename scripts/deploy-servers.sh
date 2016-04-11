#!/usr/bin/env sh

function rm_container(){
    if docker ps -f name='$1' &> /dev/null; then
        echo "Removing old container: $(docker rm -f $1) image: $(docker rmi -f $i)"
    fi
}

function build_image(){
    echo "Building $1..."
    docker build --rm -t autobd:$1 -f docker/Dockerfile.$1 .

}

build_image "server"
rm_container "autobd-server"

echo "Running server: $(docker run --net=autobd -d -p 8082:8080 -v /home/$USER/data/server-data:/home/autobd/data --name autobd-server autobd:server)"
