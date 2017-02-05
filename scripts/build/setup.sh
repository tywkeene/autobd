#!/bin/bash

#Required to install glide, just make sure
if [ ! -d "$GOPATH/bin" ]; then
    mkdir -p $GOPATH/bin
fi

if ! type glide > /dev/null; then
    curl https://glide.sh/get | sh
fi

#Try to get our deps
glide up

#build!
bash ./scripts/build/build.sh
