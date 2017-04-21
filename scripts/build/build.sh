#!/usr/bin/env bash

source ./VERSION

export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64

echo "Building autobd binary..."

go build -v \
    -ldflags "-X github.com/tywkeene/autobd/version.Version=$VERSION -X github.com/tywkeene/autobd/version.CommitHash=$COMMIT" \
    github.com/tywkeene/autobd
