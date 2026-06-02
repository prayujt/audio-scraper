#!/usr/bin/env bash
#
# scrape.sh - interactive search/select/download client for audio-scraper.
#
# Talks to the server over HTTP. Set the target with AUDIO_SCRAPER_HOST
# (default http://localhost:8080) - export it from your zshrc to point at a
# local or remote instance, e.g.:
#
#   export AUDIO_SCRAPER_HOST="http://localhost:8080"
#
# Usage:
#   scrape.sh [query...]      # query as args, or prompted if omitted
#
# Flow: search -> pick results in fzf (TAB = multi-select) -> queue download.

set -euo pipefail

HOST="${AUDIO_SCRAPER_HOST:-http://localhost:8080}"

for tool in curl jq fzf; do
	command -v "$tool" >/dev/null 2>&1 || {
		echo "missing required tool: $tool" >&2
		exit 1
	}
done

query="$*"
if [[ -z "$query" ]]; then
	read -rp "search> " query
fi
[[ -n "$query" ]] || {
	echo "no query given" >&2
	exit 1
}

echo "searching '$query' on $HOST ..." >&2
resp="$(curl -sS --get "$HOST/search" --data-urlencode "q=$query")" || {
	echo "search request failed (is the server up? AUDIO_SCRAPER_HOST=$HOST)" >&2
	exit 1
}

request_id="$(jq -r '.request_id // empty' <<<"$resp")"
if [[ -z "$request_id" ]]; then
	echo "unexpected response from server:" >&2
	echo "$resp" >&2
	exit 1
fi

mapfile -t choices < <(jq -r '.choices[]?' <<<"$resp")
if ((${#choices[@]} == 0)); then
	echo "no results for '$query'" >&2
	exit 1
fi

selected="$(printf '%s\n' "${choices[@]}" |
	fzf --multi --reverse --height=80% \
		--prompt="select> " \
		--header="TAB to multi-select, ENTER to download, ESC to cancel")" || true

if [[ -z "$selected" ]]; then
	echo "nothing selected" >&2
	exit 0
fi

# Build {request_id, choices: [...]} from the selected labels.
choices_json="$(printf '%s\n' "$selected" | jq -R . | jq -s .)"
payload="$(jq -n --arg rid "$request_id" --argjson ch "$choices_json" \
	'{request_id: $rid, choices: $ch}')"

code="$(curl -sS -o /dev/null -w '%{http_code}' \
	-X POST "$HOST/download" \
	-H 'Content-Type: application/json' \
	-d "$payload")"

count="$(printf '%s\n' "$selected" | grep -c .)"
if [[ "$code" == "202" ]]; then
	echo "queued $count selection(s) for download (HTTP $code)"
else
	echo "download request failed (HTTP $code)" >&2
	exit 1
fi
