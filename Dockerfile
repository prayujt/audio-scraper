FROM golang:1.24-alpine AS build

WORKDIR /app

COPY . .

RUN apk update && apk add -U make
RUN make build


FROM python:3.12-alpine

WORKDIR /app

COPY --from=build /app/bin/audio-scraper /app/audio-scraper
COPY ./scripts /app/scripts

RUN apk update && \
	apk add -U yt-dlp-core ffmpeg

RUN pip install ytmusicapi eyed3

EXPOSE 8080

ENTRYPOINT ["/app/audio-scraper"]
