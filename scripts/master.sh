#!/bin/bash
set -e

DOCKER=$(which docker)

if [ "$1" == "start" ]; then
    SCRIPTS_DIR=$(dirname `readlink -f $0`)
    echo "Starting registry" 2>&1
    $SCRIPTS_DIR/start-registry.sh
    echo "Starting Q4M" 2>&1
    $SCRIPTS_DIR/start-q4m.sh
elif [ "$1" == "stop" ]; then
    echo "Stopping registry" 2>&1
    $DOCKER stop kanikousen-registry
    echo "Stopping Q4M" 2>&1
    $DOCKER stop kanikousen-q4m
else
    echo "Usage: $0 start|stop" 2>&1
    exit 1
fi
