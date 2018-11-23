#!/bin/bash

wget -O docker/gobuilder/go-amd64.tar.gz "https://dl.google.com/go/go1.11.2.linux-amd64.tar.gz"
sudo docker build -t gobuilder docker/gobuilder
rm -f docker/gobuilder/go-amd64.tar.gz
