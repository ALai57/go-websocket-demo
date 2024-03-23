#!/usr/bin/env sh

# https://docs.aws.amazon.com/lambda/latest/dg/golang-package.html

SCRIPT_DIR=$(dirname "$0")
pushd $SCRIPT_DIR/..

GOOS=linux GOARCH=arm64 go build \
    -tags lambda.norpc \
    -o bootstrap \
    pkg/lambda/lambda.go

zip go-websocket.zip bootstrap

popd >/dev/null
