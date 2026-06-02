-include .env

PREFIX ?= $(HOME)/.local
BINDIR := $(PREFIX)/bin

.PHONY: all build start scrape install uninstall deploy

all: build

build:
	go build -o bin/audio-scraper cmd/*.go

start:
	./bin/audio-scraper

# Interactive search/select/download client. Pass a query with q="...":
#   make scrape q="daft punk get lucky"
scrape:
	./scripts/scrape.sh $(q)

# Install the TUI client to $(BINDIR) as audio-scrape (override with PREFIX=...).
install:
	install -d $(BINDIR)
	install -m755 scripts/scrape.sh $(BINDIR)/audio-scrape
	@echo "installed audio-scrape to $(BINDIR)"

uninstall:
	rm -f $(BINDIR)/audio-scrape
	@echo "removed audio-scrape from $(BINDIR)"

# Build, tag and push the image to docker.prayujt.com/audio-scraper.
deploy:
	./scripts/deploy.sh
