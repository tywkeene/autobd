#!/bin/bash
for i in `seq 1 $1`; do
    echo "KILL $(docker rm -f autobd-node$i)"
done
