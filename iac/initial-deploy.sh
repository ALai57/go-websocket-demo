#!/usr/bin/env sh

aws lambda create-function --function-name go-test-lambda \
--runtime provided.al2023 --handler bootstrap \
--architectures arm64 \
--role arn:aws:iam::758589815425:role/go-test-lambda-ex \
--zip-file fileb://go-test-lambda.zip
