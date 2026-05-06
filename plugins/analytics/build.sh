#!/bin/bash
# Build analytics plugin as Go plugin (.so file)

set -e

echo "Building analytics plugin..."
go build -buildmode=plugin -o analytics.so main.go

echo "Analytics plugin built successfully: analytics.so"
