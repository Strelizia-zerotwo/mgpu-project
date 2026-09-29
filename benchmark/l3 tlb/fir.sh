#!/usr/bin/env bash
set -euo pipefail
mgpu_script_dir=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
exec bash "$mgpu_script_dir/run.sh" fir "$@"
