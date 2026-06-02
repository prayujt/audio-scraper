FROM golang:1.24-alpine AS build

WORKDIR /app

COPY . .

RUN apk update && apk add -U make
RUN make build


FROM alpine:3.20

WORKDIR /app

COPY --from=build /app/bin/audio-scraper /app/audio-scraper

# Install the latest yt-dlp from pip rather than the (often months-stale) Alpine
# package, since YouTube extraction breaks quickly on old versions.
RUN apk add --no-cache ffmpeg python3 py3-pip && \
	pip install --no-cache-dir --break-system-packages -U yt-dlp

EXPOSE 8080

ENTRYPOINT ["/app/audio-scraper"]
