#!/usr/bin/env bash
#
# deploy.sh - build, tag and push the audio-scraper image to the registry.
#
# Tags the image with both the current git short SHA and "latest", then pushes
# both to docker.prayujt.com/audio-scraper.
#
# Usage:
#   scripts/deploy.sh            # build + push :<sha> and :latest
#   TAG=v1.2.3 scripts/deploy.sh # also build + push that explicit tag

set -euo pipefail

REGISTRY="docker.prayujt.com"
IMAGE="$REGISTRY/audio-scraper"

cd "$(dirname "$0")/.."

command -v docker >/dev/null 2>&1 || {
	echo "docker not found" >&2
	exit 1
}

sha="$(git rev-parse --short HEAD 2>/dev/null || echo "dev")"

tags=("$sha" "latest")
if [[ -n "${TAG:-}" ]]; then
	tags+=("$TAG")
fi

command -v ko >/dev/null 2>&1 || {
	echo "ko not found" >&2
	exit 1
}

docker build -f Dockerfile.runtime -t ko.local/audio-scraper-runtime .
export KO_DOCKER_REPO="$IMAGE"
ko build ./cmd --bare --tags="$(IFS=,; echo "${tags[*]}")"

echo "deployed $IMAGE (${tags[*]})"
