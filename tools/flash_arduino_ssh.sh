#!/bin/bash

ALLOW_SSH_AUTH_WITH_PASSWORD="-o PreferredAuthentications=password -o PasswordAuthentication=yes"

# Ensure a source argument was provided
if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
    echo "Usage: $0 <path_to_source> <ip_address_or_hostname> [username]"
    exit 1
fi

# 1. Use tinygo to build the firmware and generate a .hex file
SOURCE_PATH="$1"
TARGET_HOST="$2"
TARGET_USER="${3:-arduino}"
TARGET="$TARGET_USER@$TARGET_HOST"
HEX_OUTPUT_PATH="${SOURCE_PATH%.*}.hex"
echo "Building firmware from source '$SOURCE_PATH'..."
if ! tinygo build -o "$HEX_OUTPUT_PATH" -target=arduino-uno-q "$SOURCE_PATH"; then
    echo "Error: Failed to build firmware from $SOURCE_PATH"
    exit 1
fi

LOCAL_HEX_FILE="$HEX_OUTPUT_PATH"
HEX_FILENAME=$(basename "$LOCAL_HEX_FILE")
TARGET_HEX_PATH="/home/arduino/$HEX_FILENAME"

# 2. Upload the .hex file
echo "Uploading firmware '$HEX_FILENAME'..."
if ! scp $ALLOW_SSH_AUTH_WITH_PASSWORD "$LOCAL_HEX_FILE" "$TARGET:$TARGET_HEX_PATH"; then
    echo "Error: Failed to upload $LOCAL_HEX_FILE"
    exit 1
fi

# 3. Run the OpenOCD command on the target to program the firmware
echo "Flashing firmware..."
ssh "$TARGET" $ALLOW_SSH_AUTH_WITH_PASSWORD "/opt/openocd/bin/openocd -s /opt/openocd/share/openocd/scripts -s /opt/openocd -c \"adapter driver linuxgpiod\" -c \"adapter gpio swclk -chip 1 26\" -c \"adapter gpio swdio -chip 1 25\" -c \"adapter gpio srst -chip 1 38\" -c \"transport select swd\" -c \"adapter speed 1000\" -c \"reset_config srst_only srst_push_pull\" -f /opt/openocd/stm32u5x.cfg -c \"program $TARGET_HEX_PATH verify reset exit\""

echo "Done!"
