#!/bin/bash

# 1. Disable the arduino-router service
echo "Disabling and masking arduino-router..."
adb shell -t "sudo systemctl disable arduino-router && sudo systemctl stop arduino-router && sudo mv /etc/systemd/system/arduino-router.service /etc/systemd/system/arduino-router.service.bak && sudo systemctl mask arduino-router.service"

# 2. Install screen utility
echo "Installing screen..."
adb shell -t "sudo apt-get update && sudo apt-get install -y screen"

echo "Done!"
