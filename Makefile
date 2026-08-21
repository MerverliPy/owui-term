SHELL := /bin/sh
GO ?= go

.PHONY: all build test lint fmt clean

all: test

build:
	$(GO) build ./...

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...

fmt:
	@if [ "$$(gofmt -l .)" ]; then \
		echo "gofmt check failed: run gofmt -w on the files above"; \
		echo "$$(gofmt -l .)"; \
		exit 1; \
	fi

clean:
	rm -f owui-term

