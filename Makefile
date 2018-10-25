VERSION := $(shell sh -c 'git describe --always --tags')
LDFLAGS := -ldflags "-X main.VERSION=$(VERSION)"

all: build

build:
	mkdir -p bin
	go build -o bin/glock $(LDFLAGS) .

dist: build
	rm -rf dist/*
	mkdir -p dist/glock
	cp bin/glock dist/glock/
	tar -C dist -czvf dist/glock-$(VERSION).tar.gz glock

.PHONY= all build
