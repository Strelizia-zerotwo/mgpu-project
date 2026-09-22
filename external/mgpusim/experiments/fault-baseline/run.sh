#!/usr/bin/env bash
set -euo pipefail

mgpu_source_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
if ! command -v go >/dev/null 2>&1; then
    export PATH="/usr/local/go/bin:$PATH"
fi
mkdir -p "$HOME/mgpusim-runs"
mgpu_result_dir=$(mktemp -d "$HOME/mgpusim-runs/fault-baseline-XXXXXX")
cd "$mgpu_source_root"
git rev-parse HEAD > "$mgpu_result_dir/commit.txt"
git status --short -- . > "$mgpu_result_dir/git-status.txt"
git diff -- . > "$mgpu_result_dir/working-tree.diff"
# Includes new files and source files that upstream .gitignore accidentally hides.
tar --exclude='./.git' --exclude='*.sqlite3*' -czf "$mgpu_result_dir/source.tar.gz" .
go version > "$mgpu_result_dir/go-version.txt"
go build -o "$mgpu_result_dir/fir" ./amd/samples/fir

mgpu_common=(-timing -verify -disable-rtm -gpus=1,2 -use-unified-memory -length=16384 -report-all)
mgpu_parameters=(-vm-local-walk-cycles=100 -vm-local-walkers=8 -vm-transaction-slots=64
    -vm-max-waiters=64 -vm-host-walk-cycles=400 -vm-link-cycles=200 -vm-install-cycles=100)
for mgpu_case in demand ideal congested; do
    mgpu_mode=$mgpu_case
    mgpu_host=(-vm-host-walkers=16 -vm-host-queue=64)
    if [[ "$mgpu_case" == congested ]]; then
        mgpu_mode=demand
        mgpu_host=(-vm-host-walkers=1 -vm-host-queue=2)
    fi
    mgpu_cmd=("$mgpu_result_dir/fir" "${mgpu_common[@]}" "${mgpu_parameters[@]}"
        "${mgpu_host[@]}" "-vm-mode=$mgpu_mode" "-metric-file-name=$mgpu_result_dir/$mgpu_case")
    printf '%q ' "${mgpu_cmd[@]}" >> "$mgpu_result_dir/commands.sh"
    printf '\n' >> "$mgpu_result_dir/commands.sh"
    "${mgpu_cmd[@]}" > "$mgpu_result_dir/$mgpu_case.log" 2>&1
done
python3 experiments/fault-baseline/check.py "$mgpu_result_dir" | tee "$mgpu_result_dir/check.txt"
printf '\nResults: %s\n' "$mgpu_result_dir"
