#!/bin/bash

# Try public-key auth first (set up via tools/setup_ssh_key.sh), then fall
# back to interactive password auth if the key isn't installed yet.
ALLOW_SSH_AUTH_WITH_PASSWORD="-o PreferredAuthentications=publickey,password -o PasswordAuthentication=yes"

# Ensure a source argument was provided
if [ "$#" -lt 3 ] || [ "$#" -gt 4 ]; then
    echo "Usage: $0 <path_to_file> <target_path> <ip_address_or_hostname> [username]"
    exit 1
fi

FILE_PATH="$1"
TARGET_PATH="$2"
TARGET_HOST="$3"
TARGET_USER="${4:-arduino}"
TARGET="$TARGET_USER@$TARGET_HOST"

echo "Uploading file '$FILE_PATH'..."
if ! scp $ALLOW_SSH_AUTH_WITH_PASSWORD "$FILE_PATH" "$TARGET:$TARGET_PATH"; then
    echo "Error: Failed to upload $FILE_PATH"
    exit 1
fi

echo "Done!"
