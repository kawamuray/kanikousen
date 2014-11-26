#!/bin/bash
set -e

NAME=kanikousen-registry
STORAGE_PATH=/var/lib/docker-registry

if docker inspect $NAME >/dev/null 2>&1; then
    docker start $NAME
else
    docker run -d \
        --name $NAME \
        -v $STORAGE_PATH:/registry \
        -e STORAGE_PATH=/registry \
        -p 5000:5000 \
        registry
fi
