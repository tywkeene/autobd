#!/bin/bash

rm -rf ./vendor
bash ./scripts/static-analysis.sh
bash ./scripts/resolve-deps.sh
bash ./build.sh
