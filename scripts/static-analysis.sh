#!/bin/bash

echo "Running go vet..."
for dir in $(go list ./...); do
        go vet -v $dir
done
