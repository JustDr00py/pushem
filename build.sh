#!/bin/bash

set -e

echo "Building Pushem..."

echo "1. Building frontend..."
cd web
npm install
npm run build
cd ..

echo "2. Building backend..."
go build -o pushem cmd/server/main.go

echo ""
echo "Build complete! Run './pushem' to start the server."
