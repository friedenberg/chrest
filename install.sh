#!/usr/bin/env bash

set -eux

# (Brave uses same path as Chrome, so for Brave, say `chrome`)
OS="$(uname -s)"
BROWSER="$(echo $1 | tr '[:upper:]' '[:lower:]')"

# https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/Native_manifests#Manifest_location
# https://developer.chrome.com/extensions/nativeMessaging#native-messaging-host-location
case "$OS $BROWSER" in
    "Linux firefox")
        MANIFEST_LOCATION="$HOME/.mozilla/native-messaging-hosts";;
    "FreeBSD firefox")
        MANIFEST_LOCATION="$HOME/.mozilla/native-messaging-hosts";;
    "Darwin firefox")
        MANIFEST_LOCATION="$HOME/Library/Application Support/Mozilla/NativeMessagingHosts";;
    "Linux brave")
        MANIFEST_LOCATION="$HOME/.config/BraveSoftware/Brave-Browser/NativeMessagingHosts";;
    "Linux chrome")
        MANIFEST_LOCATION="$HOME/.config/google-chrome/NativeMessagingHosts";;
    "FreeBSD chromium")
        MANIFEST_LOCATION="$HOME/.config/chromium/NativeMessagingHosts";;
    "Linux chromium")
        MANIFEST_LOCATION="$HOME/.config/chromium/NativeMessagingHosts";;
    "Linux vivaldi")
        MANIFEST_LOCATION="$HOME/.config/vivaldi/NativeMessagingHosts";;
    "Linux edgedev")
        MANIFEST_LOCATION="$HOME/.config/microsoft-edge-dev/NativeMessagingHosts";;
    "Linux opera")
        MANIFEST_LOCATION="$HOME/.config/google-chrome/NativeMessagingHosts";;
    "Darwin chrome")
        MANIFEST_LOCATION="$HOME/Library/Application Support/Google/Chrome/NativeMessagingHosts";;
    "Darwin chromebeta")
        MANIFEST_LOCATION="$HOME/Library/Application Support/Google/Chrome Beta/NativeMessagingHosts";;
    "Darwin chromium")
        MANIFEST_LOCATION="$HOME/Library/Application Support/Chromium/NativeMessagingHosts";;
    "Darwin vivaldi")
        MANIFEST_LOCATION="$HOME/Library/Application Support/Vivaldi/NativeMessagingHosts";;
    "Darwin arc")
        MANIFEST_LOCATION="$HOME/Library/Application Support/Arc/User Data/NativeMessagingHosts";;
esac

shift

mkdir -p "$MANIFEST_LOCATION"

APP_NAME="com.linenisgreat.chrest"
pushd go
go build -o build/chrest
popd
EXE_PATH="$(pwd)/go/build/chrest"

EXTENSION_IDS="$(printf 'chrome-extension://%s/\n' "$@" | jq -R '.' | jq -s)"

case "$BROWSER" in
    chrome | chromium | chromebeta | brave | vivaldi | edgedev | opera | arc)
        MANIFEST=$(cat <<EOF
{
  "name": "$APP_NAME",
  "description": "chrest-server",
  "path": "$EXE_PATH",
  "type": "stdio",
  "allowed_origins": $EXTENSION_IDS
}
EOF
        );;
    firefox)
        MANIFEST=$(cat <<EOF
{
  "name": "$APP_NAME",
  "description": "TabFS",
  "path": "$EXE_PATH",
  "type": "stdio",
  "allowed_extensions": ["tabfs@rsnous.com"]
}
EOF
        );;
esac

echo "$MANIFEST" > "$MANIFEST_LOCATION/$APP_NAME.json"
