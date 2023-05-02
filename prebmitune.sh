#!/bin/bash
CONNECT="connect 192.168.1.171"
WAKE="input keyevent KEYCODE_WAKEUP"
HOME="input keyevent KEYCODE_HOME"

adb $CONNECT
adb $CONNECT
adb $CONNECT
adb shell $WAKE
adb shell $WAKE
#adb shell $WAKE
adb shell $HOME; sleep 2
#adb shell am start com.google.android.youtube.tvunplugged; sleep 2
#adb shell am force-stop com.google.android.youtube.tvunplugged
