#!/bin/bash

# This script allows for `core` injection, and restricted context for dockerfile.
# Since dockerfile, backend and core will be on the same temporary folder, the docker build context will be lighter

echo "==========> Building backend binary"
cd ./backend
go mod tidy
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o bin/main cmd/server/main.go
cd ..

echo "==========> Building core binary"
cd ./core
go mod tidy
cd ..

echo "==========> Creating temporary directory..."
mkdir tmp-backend
cp -r ./backend ./tmp-backend
cp -r ./core ./tmp-backend
cp ./Dockerfile.backend ./tmp-backend

echo "==========> Building backend docker image"
cd tmp-backend && docker build -t backend -f ./Dockerfile.backend .

echo "==========> Removing temporary directory..."
cd .. && rm -rf ./tmp-backend

echo "==========> Finished building backend"