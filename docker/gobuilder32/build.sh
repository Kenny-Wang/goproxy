#!/bin/bash

wget -O docker/gobuilder32/go-386.tar.gz "https://dl.google.com/go/go1.11.2.linux-386.tar.gz"
sudo docker build -t gobuilder32 docker/gobuilder32/
rm -f docker/gobuilder32/go-386.tar.gz
