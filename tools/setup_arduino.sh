#!/bin/bash

# 1. Disable the arduino-router service
echo "Disabling arduino-router..."
adb shell -t "sudo systemctl disable arduino-router"

# 2. Install screen utility
echo "Installing screen..."
adb shell -t "sudo apt-get update && sudo apt-get install -y screen"

echo "Done!"
