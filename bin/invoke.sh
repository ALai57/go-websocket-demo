#!/usr/bin/env sh

aws lambda invoke --region=us-east-1 --function-name=go-websocket output.txt
