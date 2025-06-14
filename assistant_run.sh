#!/bin/sh

cd "$(dirname "$0")" || exit 1

if [ "$(uname)" = "Darwin" ]; then
  xattr -dr com.apple.quarantine ./ffmpeg 2>/dev/null
  xattr -dr com.apple.quarantine ./ffplay 2>/dev/null
  xattr -dr com.apple.quarantine ./assistant 2>/dev/null
fi

./assistant
