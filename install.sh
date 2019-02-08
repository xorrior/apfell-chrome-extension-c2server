#!/bin/bash

# install golang
apt-get update
apt-get install golang-go

# create the directory structure
SERVERDIR="$HOME/go/src/chrome-ext-server"
mkdir -p $SERVERDIR

# copy all of the server files to the SERVERDIR path

cp -R . "$SERVERDIR"

cd "$SERVERDIR"

go get

go build -o server main.go rest.go handler.go

echo "DONE! Server binary: $SERVERDIR/server"
