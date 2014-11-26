#!/bin/bash
set -e

NAME=kanikousen-q4m
MYSQLDIR=/var/lib/kanikousen-q4m

if docker inspect $NAME >/dev/null 2>&1; then
    docker start $NAME
else
    # Install mysql to host directory, to allow data persistence
    docker run \
        -v $MYSQLDIR:/usr/local/mysql-host \
        iwata/centos6-mysql56-q4m-hs \
        cp -a /usr/local/mysql/. /usr/local/mysql-host/
    # Change binlog format to MIXED
    CT_ID=$(docker run -d \
        iwata/centos6-mysql56-q4m-hs \
        sed -i 's/^#binlog_format=mixed/binlog_format=mixed/' /etc/my.cnf)
    docker wait $CT_ID
    docker commit $CT_ID kanikousen/q4m >/dev/null
    docker run -d \
        --name $NAME \
        -v $MYSQLDIR:/usr/local/mysql-host \
        -e MYSQLDIR=/usr/local/mysql-host \
        -p 3306:3306 \
        kanikousen/q4m \
        /usr/local/mysql/bin/mysqld_safe --user=mysql >/dev/null
fi
