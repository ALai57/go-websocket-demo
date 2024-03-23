#!/usr/bin/env bash

# Usage eval `get-token.sh` will export the ACCESS_TOKEN environment variable.

if [[ -z "${KEYCLOAK_USER_NAME}" ]]; then
    echo "echo "Required KEYCLOAK_USER_NAME environment var not set. Exiting""
    exit 1
fi

if [[ -z "${KEYCLOAK_USER_PASSWORD}" ]]; then
    echo "echo "Required KEYCLOAK_USER_PASSWORD environment var not set. Exiting""
    exit 1
fi

if [[ -z "${KEYCLOAK_CLIENT_SECRET}" ]]; then
    echo "echo "Required KEYCLOAK_CLIENT_SECRET environment var not set. Exiting""
    exit 1
fi

KCHOST=https://keycloak.andrewslai.com
REALM=andrewslai
CLIENT_ID=cli-access

export KEYCLOAK_ACCESS_TOKEN=$(curl -s \
    -d "client_id=$CLIENT_ID" -d "client_secret=$KEYCLOAK_CLIENT_SECRET" \
    -d "username=$KEYCLOAK_USER_NAME" -d "password=$KEYCLOAK_USER_PASSWORD" \
    -d "grant_type=password" \
    "$KCHOST/realms/$REALM/protocol/openid-connect/token" | jq -r '.access_token')

echo "Exported KEYCLOAK_ACCESS_TOKEN '${KEYCLOAK_ACCESS_TOKEN:0:5}...' as shell environment variable"
