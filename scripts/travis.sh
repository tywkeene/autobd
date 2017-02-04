#!/bin/bash

rm -rf ./vendor
bash ./scripts/static-analysis.sh
go get github.com/golang/dep
bash ./scripts/resolve-deps.sh
bash ./build.sh
