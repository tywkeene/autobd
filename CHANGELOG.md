## Thu 16 Feb 2017 03:32:53 PM MST Version: 0.0.0
Initial re-release.

## Sat 18 Feb 2017 08:32:36 PM MST Version: 0.0.2
Added root index caching

## Sat 18 Feb 2017 08:48:29 PM MST Version: 0.0.3
Don't panic on connection/RequestIndex() fail
Updated scripts/git/generate-release.sh

## Mon 27 Feb 2017 03:03:59 PM MST Version: 0.0.4
Remove api/api.go, split it into routes/routes.go and server/server.go

## Fri 21 Apr 2017 04:08:44 PM MDT Version: 0.0.5
Updated scripts to use #!/usr/bin/env bash instead of #!/bin/bash for portability
Added CGO_ENABLED=0, GOOS=linux and GOARCH=amd64 flags to build script.
Autobd will now be a statically linked library.
