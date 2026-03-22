#!/bin/bash

# Script to update Docker Hub repository overview (full description)
# Usage: ./update_docker_overview.sh <namespace> <repository> <markdown_file>
# Environment variables: DOCKER_HUB_USERNAME, DOCKER_HUB_PASSWORD (or DOCKER_HUB_TOKEN)

set -e

NAMESPACE=$1
REPOSITORY=$2
MARKDOWN_FILE=$3

if [ -z "$NAMESPACE" ] || [ -z "$REPOSITORY" ] || [ -z "$MARKDOWN_FILE" ]; then
    echo "Usage: $0 <namespace> <repository> <markdown_file>"
    exit 1
fi

if [ ! -f "$MARKDOWN_FILE" ]; then
    echo "Error: Markdown file not found: $MARKDOWN_FILE"
    exit 1
fi

if [ -z "$DOCKER_HUB_USERNAME" ]; then
    echo "Error: DOCKER_HUB_USERNAME environment variable is not set."
    exit 1
fi

if [ -z "$DOCKER_HUB_PASSWORD" ] && [ -z "$DOCKER_HUB_TOKEN" ]; then
    echo "Error: DOCKER_HUB_PASSWORD or DOCKER_HUB_TOKEN environment variable is not set."
    exit 1
fi

# Get JWT Token
echo "Authenticating with Docker Hub..."
# If DOCKER_HUB_TOKEN is provided, use it as the password to get a session JWT
PASSWORD="${DOCKER_HUB_PASSWORD:-$DOCKER_HUB_TOKEN}"

LOGIN_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "{\"username\": \"$DOCKER_HUB_USERNAME\", \"password\": \"$PASSWORD\"}" \
    https://hub.docker.com/v2/users/login/)

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r .token)

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    echo "Error: Failed to authenticate with Docker Hub."
    echo "Response: $LOGIN_RESPONSE"
    exit 1
fi

# Read Markdown content and escape for JSON
echo "Reading markdown content..."
# We use jq to safely encode the markdown content into a JSON string
FULL_DESCRIPTION=$(jq -Rs . "$MARKDOWN_FILE")

# Update Repository Overview
echo "Updating repository overview for $NAMESPACE/$REPOSITORY..."
RESPONSE=$(curl -s -X PATCH \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "{\"full_description\": $FULL_DESCRIPTION}" \
    "https://hub.docker.com/v2/repositories/$NAMESPACE/$REPOSITORY/")

if echo "$RESPONSE" | grep -q "full_description"; then
    echo "Successfully updated Docker Hub repository overview!"
else
    echo "Error: Failed to update repository overview."
    echo "Response: $RESPONSE"
    exit 1
fi
