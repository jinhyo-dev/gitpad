#!/usr/bin/env bash
# Turn a copy of the demo repository into a repository with a conflicting
# merge in progress (for the conflict-resolution recording).
#   scripts/demo-conflict.sh <demo-repo> <target-dir>
set -euo pipefail
SRC=${1:?demo repo}; DST=${2:?target dir}
rm -rf "$DST"; cp -R "$SRC" "$DST"; cd "$DST"
git checkout -q -- . && git clean -qfd
F=api/src/main/kotlin/com/acme/api/RateLimiter.kt
cat > "$F" <<'KT'
package com.acme.api

class RateLimiter(private val limit: Int = 100) {
    fun allow(key: String): Boolean {
        val count = counter.increment(key)
        return count <= limit
    }

    fun audit(event: String) {
        log.info(event)
    }
}
KT
git add "$F" && git commit -q -m "feat(api): rate limiter baseline"
git checkout -q -b feat/enterprise-limits
sed -i.bak 's/private val limit: Int = 100/private val limit: Int = 250 \/\/ enterprise plans/; s/log.info(event)/audit.record(event, level = INFO)/' "$F" && rm -f "$F.bak"
git commit -q -am "feat(api): raise limit for enterprise, structured audit"
git checkout -q main
sed -i.bak 's/private val limit: Int = 100/private val limit: Int = config.rateLimit/; s/log.info(event)/log.info("[audit] $event")/' "$F" && rm -f "$F.bak"
git commit -q -am "feat(api): configurable rate limit"
git merge feat/enterprise-limits >/dev/null 2>&1 || true
git status --short | head -3
