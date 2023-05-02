#!/bin/bash
STOP="am force-stop com.google.android.youtube.tvunplugged; sleep 2"

#Stop Video
adb shell $STOP
adb shell input keyevent KEYCODE_SLEEP
