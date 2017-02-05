#!/bin/bash

bash ./scripts/docker/rm_container.sh "autobd-server"
bash ./scripts/docker/build_image.sh "server"

DATA_DIR="/home/$USER/data/server-data"
SECRET_DIR="/home/$USER/secret"
ETC_DIR="/home/$USER/etc/autobd"
PORT=8080

if [ ! -d "$DATA_DIR" ]; then
    mkdir -p "$DATA_DIR" && echo "Created $DATA_DIR"
fi
echo "Running server: $(docker run -d \
    --network autobd \
    -p $PORT:8080 \
    -v $DATA_DIR:/home/autobd/data \
    -v $SECRET_DIR:/home/autobd/secret \
    -v $ETC_DIR:/home/autobd/etc \
    --name autobd-server autobd:server)"
docker logs autobd-server
