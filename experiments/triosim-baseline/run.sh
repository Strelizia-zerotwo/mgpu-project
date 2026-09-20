#!/usr/bin/env bash
# Place at: mgpu-project/experiments/triosim-baseline/run.sh
# Run with: bash experiments/triosim-baseline/run.sh
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "$SCRIPT_DIR/../.." && pwd)"
SIM_DIR="$PROJECT_ROOT/external/triosim"

# Start with the supplied VGG13 trace and the README's 4-GPU example.
MODEL="${MODEL:-vgg13}"
GPU_COUNT="${GPU_COUNT:-4}"
TRACE_DIR="$SIM_DIR/sample_trace/trace2-h100-bs128/$MODEL"

if ! command -v go >/dev/null 2>&1; then
  echo "ERROR: Go is not installed or is not in PATH. Install Go, then retry." >&2
  exit 1
fi
if [[ ! "$GPU_COUNT" =~ ^[1-9][0-9]*$ ]]; then
  echo "ERROR: GPU_COUNT must be a positive integer." >&2
  exit 1
fi
if [[ "$MODEL" != vgg13 && "$MODEL" != resnet50 ]]; then
  echo "ERROR: MODEL must be vgg13 or resnet50 (bundled sample traces)." >&2
  exit 1
fi
for required in "$SIM_DIR/go.mod" "$SIM_DIR/triosim/main.go" \
  "$TRACE_DIR/tensor.csv" "$TRACE_DIR/trace.csv"; do
  if [[ ! -f "$required" ]]; then
    echo "ERROR: Missing file: $required" >&2
    echo "Place this script at experiments/triosim-baseline/run.sh and the full TrioSim source at external/triosim/." >&2
    exit 1
  fi
done

# Applies to this process only; does not change 'go env -w' settings.
# Override with: GOPROXY=https://proxy.golang.org,direct bash .../run.sh
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

mkdir -p -- "$PROJECT_ROOT/runs/triosim-baseline"
RUN_DIR="$(mktemp -d "$PROJECT_ROOT/runs/triosim-baseline/$(date +%Y%m%d-%H%M%S)-XXXXXX")"
echo "Run directory: $RUN_DIR"
printf 'RUNNING\n' > "$RUN_DIR/status.txt"
trap 'rc=$?; if (( rc != 0 )); then printf "FAILED (exit %s)\n" "$rc" > "$RUN_DIR/status.txt"; echo "ERROR: See logs in $RUN_DIR" >&2; fi' EXIT

cd -- "$SIM_DIR"
{
  date -u '+UTC: %Y-%m-%dT%H:%M:%SZ'
  printf 'Project: %s\nSimulator: %s\nTrace: %s\n' "$PROJECT_ROOT" "$SIM_DIR" "$TRACE_DIR"
  go version
  go env -json GOOS GOARCH GOPROXY GOSUMDB
  if command -v git >/dev/null 2>&1 && git -C "$PROJECT_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo 'Project commit and working tree status (local changes are not captured by the commit):'
    git -C "$PROJECT_ROOT" rev-parse HEAD || true
    git -C "$PROJECT_ROOT" status --short || true
  fi
} > "$RUN_DIR/environment.txt"

echo '[1/3] Downloading Go dependencies...'
go mod download 2>&1 | tee "$RUN_DIR/dependencies.log"

echo '[2/3] Building TrioSim...'
go build -o "$RUN_DIR/triosim" ./triosim 2>&1 | tee "$RUN_DIR/build.log"

ARGS=(
  -trace-dir "$TRACE_DIR"
  -batch-size 128
  -batch-size-sim 128
  -GPUnumber "$GPU_COUNT"
  -case 1
  -bandwidth 696
  -ptp-bandwidth 65
  -capacity 40
  -interconnects 0
)
{
  printf 'cd %q\n' "$SIM_DIR"
  printf '%q ' "$RUN_DIR/triosim" "${ARGS[@]}"
  printf '\n'
} > "$RUN_DIR/command.sh"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "$TRACE_DIR/trace.csv" "$TRACE_DIR/tensor.csv" "$RUN_DIR/triosim" > "$RUN_DIR/sha256.txt"
fi

echo "[3/3] Running $MODEL on $GPU_COUNT simulated GPUs..."
"$RUN_DIR/triosim" "${ARGS[@]}" 2>&1 | tee "$RUN_DIR/run.log"

# Verify the expected case-1 output is present. This is a smoke check,
# not a check of the paper's accuracy or the completeness of every event.
for marker in 'Estimated execution time ms,' 'Current time after AllReduce stage ms,' 'Program Execution time:'; do
  if ! grep -F "$marker" "$RUN_DIR/run.log" >> "$RUN_DIR/summary.txt"; then
    echo "ERROR: Expected output not found: $marker" >&2
    exit 1
  fi
done
printf 'PASSED (example run only)\n' > "$RUN_DIR/status.txt"
echo
echo 'Example finished. Timing summary:'
cat "$RUN_DIR/summary.txt"
echo "Logs, executable and settings: $RUN_DIR"
