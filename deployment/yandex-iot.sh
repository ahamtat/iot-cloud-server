#!/bin/bash

cd ../build/package/gocloudserver/

rm gocloudserver_*.tar.gz
./build.sh

scp gocloudserver_*.tar.gz veedo@yandex.iot:/home/veedo/Distrib
