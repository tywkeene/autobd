#!/bin/bash

source ./VERSION
echo "Building autobd binary..."
go build -v -ldflags "-X main.APIVer=$API -X main.NodeVer=$NODE -X main.CommitHash=$COMMIT" github.com/tywkeene/autobd
