#!/bin/bash
set -euo pipefail

echo "Generating protobuf stubs..."
bash scripts/generate-proto.sh

echo "Generating OpenAPI (static placeholder already present at api/openapi/openapi.yaml)"
