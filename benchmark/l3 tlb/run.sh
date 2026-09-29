#!/usr/bin/env bash
set -euo pipefail
mgpu_script_dir=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
exec python3 "$mgpu_script_dir/run.py" "$@"
