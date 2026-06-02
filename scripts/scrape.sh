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
#   scrape.sh [query...]          # search metadata -> pick -> download
#   scrape.sh replace [query...]  # search your library -> pick song ->
#                                 # pick a YouTube source -> replace its audio
#
# Download flow: search -> pick results in fzf (TAB = multi-select) -> queue.
# Replace flow:  library search -> pick one song -> pick one YouTube candidate.

set -euo pipefail

HOST="${AUDIO_SCRAPER_HOST:-http://localhost:8080}"

for tool in curl jq fzf; do
	command -v "$tool" >/dev/null 2>&1 || {
		echo "missing required tool: $tool" >&2
		exit 1
	}
done

# Replacement subcommand: re-pick the YouTube source for an existing song.
if [[ "${1:-}" == "replace" ]]; then
	shift
	query="$*"
	if [[ -z "$query" ]]; then
		read -rp "library search> " query
	fi
	[[ -n "$query" ]] || {
		echo "no query given" >&2
		exit 1
	}

	echo "searching library for '$query' on $HOST ..." >&2
	resp="$(curl -sS --get "$HOST/library/search" --data-urlencode "q=$query")" || {
		echo "library search request failed (is the server up? AUDIO_SCRAPER_HOST=$HOST)" >&2
		exit 1
	}
	request_id="$(jq -r '.request_id // empty' <<<"$resp")"
	if [[ -z "$request_id" ]]; then
		echo "unexpected response from server:" >&2
		echo "$resp" >&2
		exit 1
	fi
	mapfile -t songs < <(jq -r '.choices[]?' <<<"$resp")
	if ((${#songs[@]} == 0)); then
		echo "no library matches for '$query'" >&2
		exit 1
	fi

	song="$(printf '%s\n' "${songs[@]}" |
		fzf --reverse --height=80% --prompt="song> " \
			--header="pick the song to replace (ENTER), ESC to cancel")" || true
	[[ -n "$song" ]] || {
		echo "nothing selected" >&2
		exit 0
	}

	echo "fetching YouTube candidates ..." >&2
	cand_payload="$(jq -n --arg rid "$request_id" --arg ch "$song" '{request_id: $rid, choice: $ch}')"
	cresp="$(curl -sS -X POST "$HOST/library/candidates" \
		-H 'Content-Type: application/json' -d "$cand_payload")" || {
		echo "candidate request failed" >&2
		exit 1
	}
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

	rep_payload="$(jq -n --arg rid "$request_id" --arg ch "$candidate" '{request_id: $rid, choice: $ch}')"
	code="$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$HOST/replace" \
		-H 'Content-Type: application/json' -d "$rep_payload")"
	if [[ "$code" == "202" ]]; then
		echo "queued replacement (HTTP $code)"
	else
		echo "replace request failed (HTTP $code)" >&2
		exit 1
	fi
	exit 0
fi

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
