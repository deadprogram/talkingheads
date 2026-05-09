#!/bin/bash

ALLOW_SSH_AUTH_WITH_PASSWORD="-o PreferredAuthentications=password -o PasswordAuthentication=yes"

# Ensure arguments were provided
if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <ip_address_or_hostname> [user] [args_for_actor...]"
    echo "Example: $0 192.168.1.100 arduino --name gemmai --script gemmai.md"
    exit 1
fi

TARGET_HOST="$1"
shift

# Optional username as second argument, if it doesn't start with a dash
if [[ "$1" != -* ]] && [ "$#" -gt 0 ]; then
    TARGET_USER="$1"
    shift
else
    TARGET_USER="arduino"
fi

ACTOR_ARGS="$@"
TARGET="$TARGET_USER@$TARGET_HOST"

echo "Connecting to $TARGET to run actor with args: $ACTOR_ARGS"

# This assumes the project is already deployed to the remote machine's home directory.
# Adjust the remote path as necessary (e.g., if it's deployed to ~/talkingheads)
REMOTE_PATH="~/talkingheads"

# We pass the command over SSH. Use -t to force pseudo-terminal allocation 
# which is helpful if the process runs continuously (like an actor).
ssh -t $ALLOW_SSH_AUTH_WITH_PASSWORD "$TARGET" "\
    cd $REMOTE_PATH && \
    ./actor $ACTOR_ARGS \
"

echo "Actor stopped or connection closed."
