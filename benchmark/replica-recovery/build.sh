#!/bin/bash
# Build the traced MGPUSim samples and the analyzer into bin/.
set -e
here=$(cd "$(dirname "$0")" && pwd)
src=$here/../../external/mgpusim/amd/samples
mkdir -p "$here/bin"
apps=${*:-$(cut -d' ' -f1 "$here/apps.txt" | grep -v '^#' | sort -u)}
for a in $apps; do
	(cd "$src/$a" && go build -o "$here/bin/$a" .) &
done
(cd "$here/analyzer" && go build -o "$here/bin/rranalyze" .) &
wait
