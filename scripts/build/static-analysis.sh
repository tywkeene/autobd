#!/usr/bin/env bash

echo "Running go vet..."
go vet $(go list ./... | grep -v ./vendor)
