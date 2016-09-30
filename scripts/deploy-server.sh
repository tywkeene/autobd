#!/usr/bin/env sh

function rm_container(){
    if docker ps -f name='$1' &> /dev/null; then
        echo "Removing old container: $(docker rm -f $1)"
    fi
}

function build_image(){
    echo "Building $1..."
    docker rmi -f autobd:server
    docker build --rm -t autobd:$1 -f docker/Dockerfile.$1 .

}

rm_container "autobd-server"
build_image "server"

DATA_DIR="/home/$USER/data/server-data"
SECRET_DIR="/home/$USER/secret"
ETC_DIR="/home/$USER/etc/autobd"
PORT=8080

mkdir -p $DATA_DIR
echo "Running server: $(docker run -d \
    -p $PORT:8080 \
    -v $DATA_DIR:/home/autobd/data \
    -v $SECRET_DIR:/home/autobd/secret \
    -v $ETC_DIR:/home/autobd/etc \
    --name autobd-server autobd:server)"
docker logs autobd-server
