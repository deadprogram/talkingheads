#!/bin/bash
#
# setup_ssh_key.sh
#
# Sets up public-key (certificate-based) SSH authentication to an actor
# machine so subsequent connections no longer prompt for a password.
#
# Steps:
#   1. Ensure a local SSH key pair exists (generates an ed25519 key if not).
#   2. Copy the public key to the target's ~/.ssh/authorized_keys
#      (using ssh-copy-id when available, falling back to a manual append).
#   3. Verify that key-based login now works.
#
# Usage: setup_ssh_key.sh <ip_address_or_hostname> [username] [identity_file]
#
# Defaults:
#   username      = arduino
#   identity_file = ~/.ssh/id_ed25519

set -euo pipefail

ALLOW_SSH_AUTH_WITH_PASSWORD="-o PreferredAuthentications=password -o PasswordAuthentication=yes -o PubkeyAuthentication=no"

if [ "$#" -lt 1 ] || [ "$#" -gt 3 ]; then
    echo "Usage: $0 <ip_address_or_hostname> [username] [identity_file]"
    exit 1
fi

TARGET_HOST="$1"
TARGET_USER="${2:-arduino}"
IDENTITY_FILE="${3:-$HOME/.ssh/id_ed25519_talkingheads}"
PUBLIC_KEY="${IDENTITY_FILE}.pub"
TARGET="$TARGET_USER@$TARGET_HOST"

# 1. Generate a local key pair if one does not already exist.
if [ ! -f "$IDENTITY_FILE" ]; then
    echo "No SSH key found at $IDENTITY_FILE — generating a new ed25519 key..."
    mkdir -p "$(dirname "$IDENTITY_FILE")"
    chmod 700 "$(dirname "$IDENTITY_FILE")"
    ssh-keygen -t ed25519 -f "$IDENTITY_FILE" -N "" -C "$(whoami)@$(hostname) -> talkingheads actor"
else
    echo "Using existing SSH key at $IDENTITY_FILE"
fi

if [ ! -f "$PUBLIC_KEY" ]; then
    echo "Error: public key $PUBLIC_KEY not found"
    exit 1
fi

# 2. Install the public key on the target.
echo "Installing public key on $TARGET..."
if command -v ssh-copy-id >/dev/null 2>&1; then
    # ssh-copy-id handles permissions and de-duplication for us.
    # shellcheck disable=SC2086
    ssh-copy-id -i "$PUBLIC_KEY" $ALLOW_SSH_AUTH_WITH_PASSWORD "$TARGET"
else
    echo "ssh-copy-id not available — falling back to manual append..."
    # shellcheck disable=SC2086
    ssh $ALLOW_SSH_AUTH_WITH_PASSWORD "$TARGET" \
        "mkdir -p ~/.ssh && chmod 700 ~/.ssh && \
         touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && \
         grep -qxF \"$(cat "$PUBLIC_KEY")\" ~/.ssh/authorized_keys || \
         echo \"$(cat "$PUBLIC_KEY")\" >> ~/.ssh/authorized_keys"
fi

# 3. Verify key-based login works (disable password fallback for the test).
echo "Verifying key-based authentication..."
if ssh -i "$IDENTITY_FILE" \
       -o PreferredAuthentications=publickey \
       -o PasswordAuthentication=no \
       -o BatchMode=yes \
       "$TARGET" "echo 'key-based SSH to '\$(hostname)' OK'"; then
    echo "Done! You can now connect with: ssh $TARGET"
else
    echo "Error: key-based authentication did not work."
    exit 1
fi
