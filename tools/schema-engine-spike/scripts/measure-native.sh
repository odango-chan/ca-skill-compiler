#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
spike_dir="$(cd -- "$script_dir/.." && pwd)"
out_dir="${1:-/tmp/schema-engine-spike-native-measurements}"
mkdir -p "$out_dir"

GOTOOLCHAIN=go1.23.0 GOWORK=off go -C "$spike_dir/cmd/san" build \
  -buildvcs=false -trimpath -ldflags='-s -w' -o "$out_dir/san"

for run in 1 2 3; do
  env -u SPIKE_SUITE_ROOT -u SPIKE_SUITE_REGEX2 \
    /usr/bin/time -v -o "$out_dir/run-${run}.time" "$out_dir/san" \
    > "$out_dir/run-${run}.json"
done

printf 'Native run artifacts: %s\n' "$out_dir"
