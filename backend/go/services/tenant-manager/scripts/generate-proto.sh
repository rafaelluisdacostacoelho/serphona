#!/bin/bash

# Script to generate Go code from protobuf definitions
set -e

echo "Generating gRPC code from protobuf..."

# Create output directory if it doesn't exist
mkdir -p proto/tenantpb

# Generate Go code
protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  proto/tenant.proto

echo "✅ gRPC code generated successfully in proto/tenantpb/"
