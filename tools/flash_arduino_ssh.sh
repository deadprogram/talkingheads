#!/bin/bash

# Try public-key auth first (set up via tools/setup_ssh_key.sh), then fall
# back to interactive password auth if the key isn't installed yet.
ALLOW_SSH_AUTH_WITH_PASSWORD="-o PreferredAuthentications=publickey,password -o PasswordAuthentication=yes"

# Ensure a source argument was provided
if [ "$#" -lt 4 ] || [ "$#" -gt 5 ]; then
    echo "Usage: $0 <path_to_source> <name_of_program> <color> <ip_address_or_hostname> [username]"
    exit 1
fi

# 1. Use tinygo to build the firmware and generate a .hex file
SOURCE_PATH="$1"
PROGRAM_NAME="$2"
COLOR="$3"
TARGET_HOST="$4"
TARGET_USER="${5:-arduino}"
TARGET="$TARGET_USER@$TARGET_HOST"
HEX_OUTPUT_PATH="${PROGRAM_NAME%.*}.hex"
echo "Building firmware from source '$SOURCE_PATH'..."
cd $SOURCE_PATH
if ! tinygo build -o "$HEX_OUTPUT_PATH" -size short -target=arduino-uno-q -tags feetech -ldflags="-X main.PersonalityColor=$COLOR" .; then
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
