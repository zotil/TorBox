#!/usr/bin/env bash
# Regenerate site/ from the WordPress restored locally (tools/wp-local/docker-compose.yml, port 127.0.0.1:8080).
# Requirements: wget, Python 3 with beautifulsoup4 + lxml (see tools/requirements.txt).
#
#   tools/mirror.sh <work-directory> <wordpress-webroot>
set -euo pipefail

WORK="${1:?work directory}"
WEBROOT="${2:?webroot of the local WordPress}"
HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$HERE")"
BASE="http://127.0.0.1:8080"

rm -rf "$WORK/mirror" && mkdir -p "$WORK/mirror" && cd "$WORK/mirror"

REJECT='(\?(page_id|p|attachment_id|cat|m|s|replytocom|author|tag|feed|paged|cpage|format|rest_route)=|/feed/?($|\?)|/wp-json|xmlrpc|wp-login|wp-admin|/embed/?$|\.img\.xz|/comments/feed|#)'
# Published pages not linked from any menu: wget would not find them with --mirror alone.
SEEDS=(/ /i-want-to-build-it-from-scratch-pre-v-0-23/ /to-go/)
for s in "${SEEDS[@]}"; do
  wget --mirror --page-requisites --adjust-extension --no-host-directories -e robots=off --no-verbose \
       --tries=2 --timeout=30 --reject-regex "$REJECT" -a "$WORK/wget.log" "$BASE$s" || true
done

python3 "$HERE/postprocess.py" --mirror "$WORK/mirror" --webroot "$WEBROOT" --out "$ROOT/site"
