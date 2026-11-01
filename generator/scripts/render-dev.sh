#!/bin/sh
set -eu

config=${1:-}
if [ -z "$config" ]; then
  echo "config is required" >&2
  echo "usage: render-dev.sh <config>" >&2
  exit 1
fi

case "$config" in
  *[!a-zA-Z0-9_-]*)
    echo "invalid config: $config" >&2
    exit 1
    ;;
esac

script_directory=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
generator_directory=$(dirname "$script_directory")
config_path="$generator_directory/testdata/configs/$config.json"
dev_root=${GOILERPLATE_TEMPLATE_DEV_ROOT:-"$generator_directory/.dev"}
project="$dev_root/$config"

if [ ! -f "$config_path" ]; then
  echo "unknown config: $config" >&2
  echo "available configs:" >&2
  for available_config_path in "$generator_directory"/testdata/configs/*.json; do
    basename "$available_config_path" .json >&2
  done
  exit 1
fi

staging=$(mktemp -d)
cleanup() {
  rm -rf "$staging"
}
trap cleanup EXIT INT TERM

go -C "$generator_directory" run ./cmd/render \
  -config "$config_path" \
  -version 3.0.0-dev \
  -out "$staging"

mkdir -p "$project"
rsync --archive --checksum --delete \
  --exclude=/.env \
  --exclude=/data/ \
  --exclude=/uploads/ \
  --exclude='*_templ.go' \
  --exclude=/assets/css/output.css \
  "$staging/" "$project/"

find "$project" -type f -name '*_templ.go' -print | while IFS= read -r generated; do
  source=${generated%_templ.go}.templ
  if [ ! -f "$source" ]; then
    rm -f "$generated"
  fi
done

if [ ! -f "$project/.env" ]; then
  cp "$project/.env.example" "$project/.env"
  echo "created $project/.env"
else
  missing_env="$staging/.env.missing"
  awk '
    FNR == NR {
      if ($0 ~ /^[A-Za-z_][A-Za-z0-9_]*=/) {
        key = $0
        sub(/=.*/, "", key)
        existing[key] = 1
      }
      next
    }
    $0 ~ /^[A-Za-z_][A-Za-z0-9_]*=/ {
      key = $0
      sub(/=.*/, "", key)
      if (!(key in existing)) {
        print
      }
    }
  ' "$project/.env" "$project/.env.example" > "$missing_env"
  if [ -s "$missing_env" ]; then
    printf '\n# Added from updated template\n' >> "$project/.env"
    command cat "$missing_env" >> "$project/.env"
    echo "added new settings to $project/.env"
  fi
fi

echo "rendered $config into $project"
