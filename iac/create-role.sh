#!/usr/bin/env sh

# https://docs.aws.amazon.com/lambda/latest/dg/lambda-intro-execution-role.html
aws iam create-role \
    --role-name go-test-lambda-ex \
    --assume-role-policy-document '{"Version": "2012-10-17","Statement": [{ "Effect": "Allow", "Principal": {"Service": "lambda.amazonaws.com"}, "Action": "sts:AssumeRole"}]}'


aws iam attach-role-policy \
    --role-name go-test-lambda-ex \
    --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
