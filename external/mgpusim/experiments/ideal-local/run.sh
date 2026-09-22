#!/usr/bin/env bash
set -euo pipefail

mgpu_source_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
if ! command -v go >/dev/null 2>&1; then
    export PATH="/usr/local/go/bin:$PATH"
fi
mkdir -p "$HOME/mgpusim-runs"
mgpu_result_dir=$(mktemp -d "$HOME/mgpusim-runs/ideal-local-XXXXXX")
cd "$mgpu_source_root"
git rev-parse HEAD > "$mgpu_result_dir/commit.txt"
git diff -- . > "$mgpu_result_dir/working-tree.diff"
# Include new untracked source files as well as tracked modifications.
tar --exclude='./.git' --exclude='*.sqlite3' --exclude='*.sqlite3-*' \
    -czf "$mgpu_result_dir/source.tar.gz" .
go version > "$mgpu_result_dir/go-version.txt"
go build -o "$mgpu_result_dir/fir" ./amd/samples/fir

mgpu_common=(-timing -verify -disable-rtm -gpus=1,2 -use-unified-memory -length=1024 -report-all)
printf '%s\n' 'shared: FIR length=1024, 2 GPUs, unified allocations, 100-cycle MMU configuration' \
    'ideal: same workload, plus -ideal-local-page-table' > "$mgpu_result_dir/config.txt"
"$mgpu_result_dir/fir" "${mgpu_common[@]}" \
    -metric-file-name="$mgpu_result_dir/shared" > "$mgpu_result_dir/shared.log" 2>&1
"$mgpu_result_dir/fir" "${mgpu_common[@]}" -ideal-local-page-table \
    -metric-file-name="$mgpu_result_dir/ideal" > "$mgpu_result_dir/ideal.log" 2>&1
python3 experiments/ideal-local/check.py "$mgpu_result_dir" | tee "$mgpu_result_dir/check.txt"
printf '\nResults: %s\n' "$mgpu_result_dir"
