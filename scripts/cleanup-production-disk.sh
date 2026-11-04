#!/usr/bin/env bash
set -euo pipefail

mode="${1:-survey}"
release_dir="${2:-}"
case "$mode" in
  survey|prune) ;;
  *) echo "usage: $0 {survey|prune} RELEASE_PATH" >&2; exit 2 ;;
esac

expected="$HOME/RPBox/server/releases"
if [[ -z "$release_dir" || ! -d "$expected" || ! -d "$release_dir" ||
      "$(realpath -e "$release_dir")" != "$(realpath -e "$expected")" ]]; then
  echo 'Release path does not match the RPBox production directory' >&2
  exit 2
fi

read_pointer() {
  python3 - "$1" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as source:
    print(json.load(source)["latest_version"])
PY
}

stable="$(read_pointer "$release_dir/latest.json")"
beta="$(read_pointer "$release_dir/latest-beta.json")"
for version in "$stable" "$beta"; do
  if [[ ! "$version" =~ ^[0-9]+(\.[0-9]+)+(-[0-9]+)?$ || ! -d "$release_dir/$version" ]]; then
    echo "Invalid or missing active release: $version" >&2
    exit 2
  fi
done

mapfile -t versions < <(find "$release_dir" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' |
  grep -E '^[0-9]+(\.[0-9]+)+(-[0-9]+)?$' | LC_ALL=C sort -V)
declare -A keep=(["$stable"]=1 ["$beta"]=1)
start=$(( ${#versions[@]} > 8 ? ${#versions[@]} - 8 : 0 ))
for (( i=start; i<${#versions[@]}; i++ )); do
  keep["${versions[$i]}"]=1
done

echo "Mode: $mode; active stable: $stable; active beta: $beta"
echo 'Disk before:'
df -h /
du -sh "$release_dir" "$HOME/RPBox/server/storage/backups" 2>/dev/null || true

for version in "${versions[@]}"; do
  if [[ -z "${keep[$version]+present}" ]]; then
    echo "${mode}: inactive desktop release $version"
    if [[ "$mode" == prune ]]; then
      rm -rf -- "$release_dir/$version"
    fi
  fi
done

mapfile -t stale_uploads < <(find "$release_dir" -mindepth 1 -maxdepth 1 -type d \
  -name '.upload-*' -mtime +7 -printf '%f\n')
for name in "${stale_uploads[@]}"; do
  [[ "$name" =~ ^\.upload-[A-Za-z0-9._-]+$ ]] || continue
  echo "${mode}: stale release staging directory $name"
  if [[ "$mode" == prune ]]; then
    rm -rf -- "$release_dir/$name"
  fi
done

cache_mb() {
  if [[ -d "$1" ]]; then
    du -sm "$1" | awk '{print $1}'
  else
    echo 0
  fi
}

free_bytes="$(df -B1 --output=avail / | tail -n 1 | tr -d ' ')"
if [[ "$mode" == prune ]] && (( free_bytes < 3 * 1024 * 1024 * 1024 )); then
  if pgrep -f '(^|/)go build|(^|/)npm (ci|install)' >/dev/null; then
    echo 'Build in progress; package caches retained'
  else
    if (( $(cache_mb "$HOME/.npm/_cacache") > 128 )) && command -v npm >/dev/null; then
      npm cache clean --force
    fi
    if (( $(cache_mb "$HOME/go/pkg/mod") > 256 || $(cache_mb "$HOME/.cache/go-build") > 128 )) && command -v go >/dev/null; then
      go clean -cache -modcache
    fi
    if (( $(cache_mb "$HOME/.cache/pip") > 64 )) && python3 -m pip --version >/dev/null 2>&1; then
      python3 -m pip cache purge
    fi
  fi
fi

test -d "$release_dir/$stable"
test -d "$release_dir/$beta"
echo 'Disk after:'
df -h /
du -sh "$release_dir" "$HOME/RPBox/server/storage/backups" 2>/dev/null || true
free_bytes="$(df -B1 --output=avail / | tail -n 1 | tr -d ' ')"
if [[ "$mode" == prune ]] && (( free_bytes < 2 * 1024 * 1024 * 1024 )); then
  echo 'Less than 2 GiB remains after safe cleanup; inspect backups and expand storage' >&2
  exit 1
fi
