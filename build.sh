#!/bin/bash
set -e

echo "Building shlx..."

# Build frontend
echo "Building frontend..."
cd frontend
npm install
npm run build
cd ..

# Build backend
echo "Building backend..."
go mod download
go build -o shlx .

echo "Build complete! Run './shlx' to start the server."
