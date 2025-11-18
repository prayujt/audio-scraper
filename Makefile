-include .env

.PHONY: all build start

all: build

build:
	go build -o bin/audio-scraper cmd/*.go

start:
	./bin/audio-scraper
