#!/bin/bash

source ./VERSION
echo "Building autobd binary..."
go build -v -ldflags "-X github.com/tywkeene/autobd/version.Version=$VERSION -X github.com/tywkeene/autobd/version.CommitHash=$COMMIT" github.com/tywkeene/autobd
