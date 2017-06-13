#!/usr/bin/env bash

echo "Running go vet..."
go vet $(glide novendor)
