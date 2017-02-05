#!/bin/bash

rm -rf ./vendor
bash ./scripts/build/static-analysis.sh
bash ./scripts/build/setup.sh
