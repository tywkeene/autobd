#!/usr/bin/env bash

rm_container(){
    if docker ps -f name='$1' &> /dev/null; then
        echo "Removing old container: $(docker rm -f $1)"
    fi
}

rm_container $1
