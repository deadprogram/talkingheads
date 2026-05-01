#!/bin/bash

# Opens a serial console to the Arduino Q board via adb.
# Uses screen on the remote device at 115200 baud.

echo "Connecting to /dev/ttyHS1 at 115200 baud..."

SCREENRC='hardstatus alwayslastline "Serial: /dev/ttyHS1 @ 115200 | Exit: Ctrl+A then K"'
exec adb shell -t "echo '$SCREENRC' > /tmp/.screenrc_serial && screen -c /tmp/.screenrc_serial /dev/ttyHS1 115200"
