#!/bin/sh
set -eu

TEMPLATE_PATH="${CONFIG_TEMPLATE:-/app/config/config.prod.yaml}"
TARGET_PATH="${CONFIG_PATH:-/app/config/config.yaml}"

if [ ! -f "$TEMPLATE_PATH" ]; then
  echo "Config template not found: $TEMPLATE_PATH" >&2
  exit 1
fi

envsubst < "$TEMPLATE_PATH" > "$TARGET_PATH"

exec "$@"
