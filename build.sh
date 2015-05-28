#!/bin/bash
go build -v -a -ldflags "-X main.commit $(git rev-parse --short=10 HEAD)" github.com/tywkeene/autobd
