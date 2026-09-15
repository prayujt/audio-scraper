FROM golang:1.24-alpine AS build

WORKDIR /app

COPY . .

RUN apk update && apk add -U make
RUN make build


FROM alpine:3.20

WORKDIR /app

COPY --from=build /app/bin/audio-scraper /app/audio-scraper

# Install the latest yt-dlp from pip rather than the (often months-stale) Alpine
# package, since YouTube extraction breaks quickly on old versions. Deno and
# yt-dlp-ejs provide the JavaScript challenge runtime; bgutil-ytdlp-pot-provider
# lets yt-dlp use the pod-local PO-token sidecar at 127.0.0.1:4416.
RUN apk add --no-cache ffmpeg python3 py3-pip deno && \
	pip install --no-cache-dir --break-system-packages -U \
		yt-dlp \
		yt-dlp-ejs \
		"bgutil-ytdlp-pot-provider==2.0.0" && \
	printf '%s\n' \
		'--js-runtimes deno' \
		'--remote-components' \
		'ejs:github' \
		> /etc/yt-dlp.conf

EXPOSE 8080

ENTRYPOINT ["/app/audio-scraper"]
