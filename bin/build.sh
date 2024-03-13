#!/usr/bin/env sh

# https://docs.aws.amazon.com/lambda/latest/dg/golang-package.html
GOOS=linux GOARCH=arm64 go build \
    -tags lambda.norpc \
    -o bootstrap \
    pkg/main.go

zip go-test-lambda.zip bootstrap
