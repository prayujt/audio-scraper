#!/usr/bin/env bash
#
# curated.sh - curated download client for audio-scraper.
#
# Search iTunes for metadata, then manually pick the YouTube source for each
# track before downloading.
#
# Talks to the server over HTTP. Set the target with AUDIO_SCRAPER_HOST
# (default http://localhost:8080).
#
# Usage:
#   curated.sh [query...]   # search iTunes -> pick track ->
#                           # pick a YouTube source -> download

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

echo "searching for '$query' on $HOST ..." >&2
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

# Filter to Track: entries only.
mapfile -t tracks < <(jq -r '.choices[]?' <<<"$resp" | grep '^Track:')
if ((${#tracks[@]} == 0)); then
	echo "no track matches for '$query'" >&2
	exit 1
fi

track="$(printf '%s\n' "${tracks[@]}" |
	fzf --reverse --height=80% --prompt="track> " \
		--header="pick the track to download (ENTER), ESC to cancel")" || true
[[ -n "$track" ]] || {
	echo "nothing selected" >&2
	exit 0
}

echo "fetching YouTube candidates ..." >&2
cand_payload="$(jq -n --arg rid "$request_id" --arg ch "$track" '{request_id: $rid, choice: $ch}')"
cresp="$(curl -sS -X POST "$HOST/curated/candidates" \
	-H 'Content-Type: application/json' -d "$cand_payload")" || {
	echo "candidate request failed" >&2
	exit 1
}
cand_request_id="$(jq -r '.request_id // empty' <<<"$cresp")"
if [[ -z "$cand_request_id" ]]; then
	echo "unexpected response from server:" >&2
	echo "$cresp" >&2
	exit 1
fi
mapfile -t candidates < <(jq -r '.choices[]?' <<<"$cresp")
if ((${#candidates[@]} == 0)); then
	echo "no YouTube candidates found" >&2
	exit 1
fi

candidate="$(printf '%s\n' "${candidates[@]}" |
	fzf --reverse --height=80% --prompt="youtube> " \
		--header="pick the correct source (ENTER), ESC to cancel")" || true
[[ -n "$candidate" ]] || {
	echo "nothing selected" >&2
	exit 0
}

dl_payload="$(jq -n --arg rid "$cand_request_id" --arg ch "$candidate" '{request_id: $rid, choice: $ch}')"
code="$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$HOST/curated/download" \
	-H 'Content-Type: application/json' -d "$dl_payload")"
if [[ "$code" == "202" ]]; then
	echo "queued curated download (HTTP $code)"
else
	echo "download request failed (HTTP $code)" >&2
	exit 1
fi
