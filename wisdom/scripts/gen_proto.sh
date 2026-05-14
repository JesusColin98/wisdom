#!/usr/bin/env bash
# gen_proto.sh — Generate gRPC Go stubs for all Wisdom services.
#
# Prerequisites:
#   brew install protobuf
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
#
# Usage:
#   ./scripts/gen_proto.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
PROTO_DIR="${ROOT_DIR}/proto"
OUT_DIR="${ROOT_DIR}/pkg"

echo "==> Generating gRPC stubs from ${PROTO_DIR}"

PROTOS=(
  cortex.proto
  thalamus.proto
  mastery.proto
  researcher.proto
  curriculum.proto
  integrations.proto
  entity.proto
  pubsub_events.proto
)

for proto in "${PROTOS[@]}"; do
  echo "  -> ${proto}"
  protoc \
    --proto_path="${PROTO_DIR}" \
    --go_out="${OUT_DIR}" \
    --go_opt=paths=source_relative \
    --go-grpc_out="${OUT_DIR}" \
    --go-grpc_opt=paths=source_relative \
    "${PROTO_DIR}/${proto}"
done

echo "==> Done. Generated stubs in ${OUT_DIR}"
