#!/bin/bash
for i in `seq 1 $1`; do
    docker rm -f autobd-node$i
done
