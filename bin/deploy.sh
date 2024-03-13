#!/usr/bin/env sh

aws lambda update-function-code --function-name go-test-lambda-tf \
--zip-file fileb://go-test-lambda.zip
