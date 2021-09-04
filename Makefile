#!/bin/bash

SHELL             = /bin/bash
PLATFORMS        ?= linux/arm/v5 linux/arm/v6 linux/arm/v7 linux/arm64 linux/amd64 darwin/amd64 windows/amd64
IMAGE_PREFIX     ?= igolaizola
COMMIT_SHORT     ?= $(shell git rev-parse --verify --short HEAD)
VERSION          ?= $(COMMIT_SHORT)
VERSION_NOPREFIX ?= $(shell echo $(VERSION) | sed -e 's/^[[v]]*//')

.PHONY: app-build
app-build:
	@for platform in $(PLATFORMS) ; do \
		os=$$(echo $$platform | cut -f1 -d/); \
		arch=$$(echo $$platform | cut -f2 -d/); \
		arm=$$(echo $$platform | cut -f3 -d/); \
		arm=$${arm#v}; \
		ext=""; \
		if [ "$$os" == "windows" ]; then \
			ext=".exe"; \
		fi; \
		file=./bin/zeken-$(VERSION_NOPREFIX)-$$(echo $$platform | tr / -)$$ext; \
		GOOS=$$os GOARCH=$$arch GOARM=$$arm CGO_ENABLED=0 \
		go build \
			-a -x -tags netgo,timetzdata -installsuffix cgo -installsuffix netgo \
			-ldflags " \
				-X main.Version=$(VERSION_NOPREFIX) \
				-X main.GitRev=$(COMMIT_SHORT) \
			" \
			-o $$file \
			./cmd/zeken; \
		if [ $$? -ne 0 ]; then \
			exit 1; \
		fi; \
		chmod +x $$file; \
	done

.PHONY: docker-build
docker-build:
	@platforms=($(PLATFORMS)); \
	platform=$${platforms[0]}; \
	if [[ $${#platforms[@]} -ne 1 ]]; then \
    	echo "Multi-arch build not supported"; \
		exit 1; \
	fi; \
	docker build --platform $$platform -t $(IMAGE_PREFIX)/zeken:$(VERSION) .; \
	if [ $$? -ne 0 ]; then \
		exit 1; \
	fi

.PHONY: docker-buildx
docker-buildx:
	@platforms=($(PLATFORMS)); \
	platform=$$(IFS=, ; echo "$${platforms[*]}"); \
	docker buildx build --platform $$platform -t $(IMAGE_PREFIX)/zeken:$(VERSION) .

.PHONY: clean
clean:
	rm -rf bin
