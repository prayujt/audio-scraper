#!/usr/bin/env bash
#
# playlist.sh - import a public Spotify playlist into your library.
#
# Give it an open.spotify.com playlist link; the server creates a Subsonic
# playlist named after the Spotify playlist (unless one already exists),
# downloads the first iTunes match for each track, and fills in the playlist
# incrementally as Navidrome indexes them.
#
# Talks to the server over HTTP. Set the target with AUDIO_SCRAPER_HOST
# (default http://localhost:8080) - export it from your zshrc to point at a
# local or remote instance, e.g.:
#
#   export AUDIO_SCRAPER_HOST="http://localhost:8080"
#
# Usage:
#   playlist.sh [spotify-playlist-url]

set -euo pipefail

HOST="${AUDIO_SCRAPER_HOST:-http://localhost:8080}"

for tool in curl jq; do
	command -v "$tool" >/dev/null 2>&1 || {
		echo "missing required tool: $tool" >&2
		exit 1
	}
done

url="${1:-}"
if [[ -z "$url" ]]; then
	read -rp "spotify playlist url> " url
fi
[[ -n "$url" ]] || {
	echo "no playlist url given" >&2
	exit 1
}

echo "importing playlist from $HOST ..." >&2
payload="$(jq -n --arg u "$url" '{playlist_url: $u}')"
body="$(mktemp)"
trap 'rm -f "$body"' EXIT
code="$(curl -sS -o "$body" -w '%{http_code}' -X POST "$HOST/import" \
	-H 'Content-Type: application/json' -d "$payload")" || {
	echo "import request failed (is the server up? AUDIO_SCRAPER_HOST=$HOST)" >&2
	exit 1
}

case "$code" in
202)
	name="$(jq -r '.name // empty' <"$body")"
	queued="$(jq -r '.queued // 0' <"$body")"
	echo "queued $queued track(s) into playlist '$name'"
	;;
409)
	echo "playlist already exists" >&2
	exit 1
	;;
*)
	echo "import request failed (HTTP $code)" >&2
	cat "$body" >&2
	exit 1
	;;
esac
