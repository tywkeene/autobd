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

build_image "staging"
build_image "dev"
build_image "deploy"

rm_container "autobd-staging"
rm_container "autobd-dev"
rm_container "autobd-deploy"

echo "Running staging: $(docker run --net=frontend -d -p 8080:8080 -v /home/$USER/data/server-data:/home/autobd-staging/data --name autobd-staging autobd:staging)"
echo "Running dev: $(docker run --net=frontend -d -p 8081:8080 -v /home/$USER/data/server-data:/home/autobd-dev/data --name autobd-dev autobd:dev)"
echo "Running deploy: $(docker run --net=frontend -d -p 8082:8080 -v /home/$USER/data/server-data:/home/autobd/data --name autobd-deploy autobd:deploy)"
