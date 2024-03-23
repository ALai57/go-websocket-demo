#!/usr/bin/env sh

SCRIPT_DIR=$(dirname "$0")
pushd $SCRIPT_DIR/..

aws lambda update-function-code --function-name go-websocket \
    --zip-file fileb://go-websocket.zip

popd >/dev/null
