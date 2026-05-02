#!/bin/bash

ALLOW_SSH_AUTH_WITH_PASSWORD="-o PreferredAuthentications=password -o PasswordAuthentication=yes"

# Ensure a source argument was provided
if [ "$#" -lt 3 ] || [ "$#" -gt 4 ]; then
    echo "Usage: $0 <path_to_source> <name_of_program> <ip_address_or_hostname> [username]"
    exit 1
fi

# 1. Use go to build the program
SOURCE_PATH="$1"
PROGRAM_NAME="$2"
TARGET_HOST="$3"
TARGET_USER="${4:-arduino}"
TARGET="$TARGET_USER@$TARGET_HOST"
OUTPUT_PATH="${PROGRAM_NAME%.*}"
echo "Building program from source '$SOURCE_PATH'..."
if ! GOOS=linux GOARCH=arm64 go build -o "$OUTPUT_PATH" "$SOURCE_PATH"; then
    echo "Error: Failed to build program from $SOURCE_PATH"
    exit 1
fi

LOCAL_FILE="$OUTPUT_PATH"
FILE_NAME=$(basename "$LOCAL_FILE")
TARGET_FILE_PATH="/home/arduino/$FILE_NAME"

# 2. Upload the file
echo "Uploading program '$FILE_NAME'..."
if ! scp $ALLOW_SSH_AUTH_WITH_PASSWORD "$LOCAL_FILE" "$TARGET:$TARGET_FILE_PATH"; then
    echo "Error: Failed to upload $LOCAL_FILE"
    exit 1
fi

echo "Done!"
