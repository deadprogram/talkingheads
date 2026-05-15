#!/bin/bash

# Try public-key auth first (set up via tools/setup_ssh_key.sh), then fall
# back to interactive password auth if the key isn't installed yet.
ALLOW_SSH_AUTH_WITH_PASSWORD="-o PreferredAuthentications=publickey,password -o PasswordAuthentication=yes"

# Ensure a source argument was provided
if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
    echo "Usage: $0 <ip_address_or_hostname> [username]"
    exit 1
fi

# connect via ssh to the target and send a shutdown command
TARGET_HOST="$1"
TARGET_USER="${2:-arduino}"
TARGET="$TARGET_USER@$TARGET_HOST"
echo "Shutting down $TARGET..."
ssh -t "$TARGET" $ALLOW_SSH_AUTH_WITH_PASSWORD "sudo shutdown now"

echo "Done!"
