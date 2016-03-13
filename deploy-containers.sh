#!/usr/bin/env sh

function rm_container(){
    if docker ps -f name='$1' &> /dev/null; then
        echo "Removing old $(docker rm -f $1)"
    fi
}

echo "Building staging..."
docker build --rm -t autobd:staging -f docker/Dockerfile.staging .
echo "Building dev..."
docker build --rm -t autobd:dev -f docker/Dockerfile.dev .
echo "Building deploy..."
docker build --rm -t autobd:deploy -f docker/Dockerfile.deploy .

rm_container "autobd-staging"
rm_container "autobd-dev"
rm_container "autobd-deploy"

echo "Running staging: $(docker run -d -p 8080:8080 -v /home/$USER/data:/home/autobd-staging/data --name autobd-staging autobd:staging)"
echo "Running dev: $(docker run -d -p 8081:8080 -v /home/$USER/data:/home/autobd-dev/data --name autobd-dev autobd:dev)"
echo "Running deploy: $(docker run -d -p 8082:8080 -v /home/$USER/data:/home/autobd/data --name autobd-deploy autobd:deploy)"
