#!/bin/bash

rm -rf ./vendor
bash ./scripts/static-analysis.sh
bash ./scripts/bootstrap-build.sh
