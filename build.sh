#!/bin/bash

source ./VERSION
go build -v -ldflags "-X main.ServerVer=$SERVER -X main.CommitHash=$COMMIT" github.com/tywkeene/autobd
