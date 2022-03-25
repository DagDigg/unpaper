#!/bin/bash

# This script allows for `core` injection, and restricted context for dockerfile.
# Since dockerfile, backend and core will be on the same temporary folder, the docker build context will be lighter
echo "==========> Creating temporary directory..."
mkdir tmp-extauth
cd tmp-extauth

cp -r ../extauth .
cp -r ../core .
cp ../Dockerfile.extauth .

echo "==========> Building extauth binary"
cd ./extauth
go mod tidy
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o bin/main main.go
cd ..

echo "==========> Building core binary"
cd ./core
go mod tidy
cd ..

echo "==========> Building extauth docker image"
docker build -t extauth -f ./Dockerfile.extauth .

echo "==========> Removing temporary directory..."
cd .. && rm -rf ./tmp-extauth

echo "==========> Finished building extauth"