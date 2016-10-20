#!/bin/sh

VERSION=1.0.2

# build for a Linux target, as it will be deployed inside an Ubuntu container:
GOPATH=`pwd` GOOS=linux GOARCH=amd64 go build rancher-configurator.go

# build and push container:
docker build -t odoko/rancher-configurator:$VERSION .
docker push odoko/rancher-configurator:$VERSION
