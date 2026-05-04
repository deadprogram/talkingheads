#!/bin/bash

ALLOW_SSH_AUTH_WITH_PASSWORD="-o PreferredAuthentications=password -o PasswordAuthentication=yes"

# Ensure a source argument was provided
if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
    echo "Usage: $0 <path_to_file> <ip_address_or_hostname> [username]"
    exit 1
fi

FILE_PATH="$1"
TARGET_HOST="$2"
TARGET_USER="${3:-arduino}"
TARGET="$TARGET_USER@$TARGET_HOST"

echo "Uploading file '$FILE_PATH'..."
if ! scp $ALLOW_SSH_AUTH_WITH_PASSWORD "$FILE_PATH" "$TARGET:/home/$TARGET_USER/"; then
    echo "Error: Failed to upload $FILE_PATH"
    exit 1
fi

echo "Done!"
