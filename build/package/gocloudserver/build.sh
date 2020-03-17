#!/bin/bash

DirName=gocloudserver_$(date +%Y%m%d)

# Build executable
mkdir $DirName
mkdir $DirName/cmd
mkdir $DirName/cmd/gocloudserver
cd $DirName/cmd/gocloudserver
go build ../../../../../../cmd/gocloudserver/gocloudserver.go
#strip ./gocloudserver

cd ../../..

# Copy configure
mkdir $DirName/configs
cp ../../../configs/gocloudserver.yaml $DirName/configs

# Make archive
tar -zcvf $DirName.tar.gz $DirName

# Delete temporary files
rm -fR $DirName