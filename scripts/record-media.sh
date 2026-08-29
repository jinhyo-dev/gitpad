#!/usr/bin/env bash
# Record every docs/tapes/*.tape into docs/media/*.gif (+ docs/screenshots/*.png).
# Requires vhs (brew install vhs). Usage: scripts/record-media.sh [tape-name…]
set -euo pipefail
cd "$(dirname "$0")/.."
DEMO=${DEMO_DIR:-/tmp/gitpad-demo}
CONFLICT=${CONFLICT_DIR:-/tmp/gitpad-conflict}
go build -o gitpad .
go run ./scripts/demo-repo "$DEMO" >/dev/null
scripts/demo-conflict.sh "$DEMO" "$CONFLICT" >/dev/null
export GITPAD_DEMO="$DEMO" GITPAD_CONFLICT="$CONFLICT" PATH="$PWD:$PATH"
names=("$@")
[ ${#names[@]} -eq 0 ] && names=(log menu commit push rebase conflicts palette undo search)
for n in "${names[@]}"; do
  echo "▶ $n"
  # every tape starts from a pristine demo repo
  go run ./scripts/demo-repo "$DEMO" >/dev/null
  vhs "docs/tapes/$n.tape" >/dev/null
done
ls -la docs/media docs/screenshots
