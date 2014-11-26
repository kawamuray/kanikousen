#!/bin/bash
set -e

MASTER_IP=172.17.42.1
DOCKER_IP=$MASTER_IP

NAME=kanikousen-worker

if docker inspect $NAME >/dev/null 2>&1; then
    docker start $NAME
else
    # TODO use environment variables instead
    docker run -d \
        --name $NAME \
        -v /var/log/kanikousen-worker:/var/log \
        kanikousen/worker \
        -b "q4m:dsn=root@tcp($MASTER_IP:3306)/kanikousen" \
        -d "tcp://$DOCKER_IP:2375" \
        -r "$MASTER_IP:5000" \
        -o /var/log/kanikousen-worker.log \
        >/dev/null
fi
