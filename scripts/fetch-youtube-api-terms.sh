#!/usr/bin/env bash
# Download YouTube API legal/docs pages into local snapshots/ and update manifest hashes.
# Does not commit Google HTML; only manifest.json is meant for git.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIR="$ROOT/docs/legal/youtube-api"
MANIFEST="$DIR/manifest.json"
SNAP="$DIR/snapshots"
UA="WonderfeedLegalRefresh/1.0 (+https://github.com/behaviorengineering/wonderfeed; terms tracking)"

if [[ ! -f "$MANIFEST" ]]; then
  echo "missing manifest: $MANIFEST" >&2
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi

mkdir -p "$SNAP"

python3 - "$MANIFEST" "$SNAP" "$UA" <<'PY'
import hashlib
import json
import os
import subprocess
import sys
from datetime import datetime, timezone

manifest_path, snap_dir, ua = sys.argv[1], sys.argv[2], sys.argv[3]

with open(manifest_path, encoding="utf-8") as f:
    manifest = json.load(f)

prev = {
    d["id"]: d.get("sha256")
    for d in manifest.get("documents", [])
    if d.get("id")
}

changed = []
now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

for doc in manifest["documents"]:
    doc_id = doc["id"]
    url = doc["url"]
    dest = os.path.join(snap_dir, f"{doc_id}.md")
    result = subprocess.run(
        [
            "curl",
            "-fsSL",
            "--max-time",
            "60",
            "-A",
            ua,
            "-H",
            "Accept: text/html,application/xhtml+xml",
            "-o",
            dest,
            url,
        ],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(f"FAILED   {doc_id}: {result.stderr.strip() or result.stdout.strip()}", file=sys.stderr)
        sys.exit(1)
    with open(dest, "rb") as fh:
        body = fh.read()
    if len(body) < 500:
        print(f"FAILED   {doc_id}: response too small ({len(body)} bytes)", file=sys.stderr)
        sys.exit(1)
    digest = hashlib.sha256(body).hexdigest()
    old = prev.get(doc_id)
    doc["fetched_at"] = now
    doc["sha256"] = digest
    doc["bytes"] = len(body)
    status = "same" if old == digest else ("new" if not old else "CHANGED")
    if status == "CHANGED":
        changed.append(doc_id)
    print(f"{status:8}  {doc_id}  {digest[:12]}…  {len(body)} bytes  -> {dest}")

manifest["updated_at"] = now
with open(manifest_path, "w", encoding="utf-8") as f:
    json.dump(manifest, f, indent=2)
    f.write("\n")

print("")
if changed:
    print("Hash changes (review revision history, then commit manifest.json):")
    for doc_id in changed:
        print(f"  - {doc_id}")
elif any(prev.values()):
    print("No hash changes versus previous manifest.")
else:
    print("Initial hashes written to manifest.json.")
PY
