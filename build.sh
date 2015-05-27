#!/bin/bash
gb build -a -ldflags "-X main.commit $(git rev-parse --short=10 HEAD)" github.com/SaviorPhoenix/autobd
