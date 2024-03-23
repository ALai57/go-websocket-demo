#!/usr/bin/env sh

# Production requires the use of an API key
wscat -H 'x-api-key: $WEBSOCKET_API_KEY' \
    -c wss://ccsvtpq2pb.execute-api.us-east-1.amazonaws.com/prod/
