#!/bin/bash

ALLOW_SSH_AUTH_WITH_PASSWORD="-o PreferredAuthentications=password -o PasswordAuthentication=yes"

# Ensure a source argument was provided
if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
    echo "Usage: $0 <ip_address_or_hostname> [username]"
    exit 1
fi

# connect via ssh to the target and open a shell
TARGET_HOST="$1"
TARGET_USER="${2:-arduino}"
TARGET="$TARGET_USER@$TARGET_HOST"
echo "Connecting to $TARGET..."
ssh "$TARGET" $ALLOW_SSH_AUTH_WITH_PASSWORD

echo "Done!"
