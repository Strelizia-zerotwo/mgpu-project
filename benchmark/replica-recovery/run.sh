#!/bin/bash
# usage: run.sh APP TAG ARGS...
# Runs APP in emulation on 4 unified GPUs with unified memory and MGPU_MEMTRACE,
# then analyzes the trace. Output: results/replica-recovery/APP/TAG/.
here=$(cd "$(dirname "$0")" && pwd)
app=$1; tag=$2; shift 2
out=$here/../../results/replica-recovery/$app/$tag
mkdir -p "$out" && cd "$out" || exit 1
echo "$app $*" > cmd.txt
start=$(date +%s)
MGPU_MEMTRACE=$out/trace timeout ${TIMEOUT:-3600} "$here/bin/$app" \
	-unified-gpus=1,2,3,4 -use-unified-memory -verify "$@" > sim.log 2>&1
rc=$?
echo "sim rc=$rc $(( $(date +%s) - start ))s $(grep -aE 'Passs?ed|Fail|Mismatch|panic' sim.log | head -1)" > status.txt
rm -f akita_sim_*.sqlite3
"$here/bin/rranalyze" -trace trace -pages ${PAGES:-4096,65536,2097152} -json result.json > summary.txt 2>&1
echo "ana rc=$? $(( $(date +%s) - start ))s" >> status.txt
[ -n "$KEEP_TRACE" ] || rm -f trace.bin
cat status.txt
