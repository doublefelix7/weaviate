#!/usr/bin/env bash

GEN_DIR=./grpc/generated
OUT_DIR="$GEN_DIR/protocol"

echo "Generating Go protocol stubs..."

rm -fr $OUT_DIR && mkdir -p $OUT_DIR && cd $GEN_DIR && protoc \
    --proto_path=../proto \
    --go_out=paths=source_relative:protocol \
    --go-grpc_out=paths=source_relative:protocol \
    ../proto/*.proto

cd - && go run ./tools/license_headers/main.go

goimports -w $OUT_DIR

# running gofumpt twice is on purpose
# it doesn't work for the first time only after second run the formatting is proper
gofumpt -w $OUT_DIR
gofumpt -w $OUT_DIR

echo "Success"
