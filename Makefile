-include .env

PREFIX ?= $(HOME)/.local
BINDIR := $(PREFIX)/bin

.PHONY: all build start scrape replace playlist install uninstall deploy

all: build

build:
	go build -o bin/audio-scraper cmd/*.go

start:
	./bin/audio-scraper

# Interactive search/select/download client. Pass a query with q="...":
#   make scrape q="daft punk get lucky"
scrape:
	./scripts/scrape.sh $(q)

# Interactive audio replacement client. Pass a query with q="...":
#   make replace q="daft punk get lucky"
replace:
	./scripts/replace.sh $(q)

# Spotify playlist import client. Pass the link with url="...":
#   make playlist url="https://open.spotify.com/playlist/..."
playlist:
	./scripts/playlist.sh $(url)

# Install the TUI clients to $(BINDIR) (override with PREFIX=...):
#   audio-scrape, audio-replace, audio-playlist
install:
	install -d $(BINDIR)
	install -m755 scripts/scrape.sh $(BINDIR)/audio-scrape
	install -m755 scripts/replace.sh $(BINDIR)/audio-replace
	install -m755 scripts/playlist.sh $(BINDIR)/audio-playlist
	@echo "installed audio-scrape, audio-replace, audio-playlist to $(BINDIR)"

uninstall:
	rm -f $(BINDIR)/audio-scrape $(BINDIR)/audio-replace $(BINDIR)/audio-playlist
	@echo "removed audio-scrape, audio-replace, audio-playlist from $(BINDIR)"

# Build, tag and push the image to docker.prayujt.com/audio-scraper.
deploy:
	./scripts/deploy.sh
