FROM golang:1.24-alpine AS build

WORKDIR /app

COPY . .

RUN apk update && apk add make
RUN make build

FROM alpine:latest

WORKDIR /app

COPY --from=build /app/bin/audio-scraper /app/audio-scraper

EXPOSE 8080

ENTRYPOINT ["/app/audio-scraper"]
