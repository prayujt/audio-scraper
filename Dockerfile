FROM golang:1.24-alpine AS build

WORKDIR /app

COPY . .

RUN apk update && apk add -U make
RUN make build


FROM alpine:3.20

WORKDIR /app

COPY --from=build /app/bin/audio-scraper /app/audio-scraper

RUN apk update && \
	apk add -U yt-dlp ffmpeg

EXPOSE 8080

ENTRYPOINT ["/app/audio-scraper"]
