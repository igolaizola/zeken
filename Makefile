#!/bin/bash
SHELL =  /bin/bash

.PHONY: all
all:
	rm -rf bin
	mkdir bin
	GOOS=linux GOARCH=arm GOARM=5 go build -o bin/zeken-arm5 cmd/zeken/main.go
	GOOS=linux GOARCH=arm GOARM=6 go build -o bin/zeken-arm6 cmd/zeken/main.go
	GOOS=linux GOARCH=arm GOARM=7 go build -o bin/zeken-arm7 cmd/zeken/main.go
	GOOS=linux GOARCH=arm64 go build -o bin/zeken-arm64 cmd/zeken/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/zeken-linux cmd/zeken/main.go
	GOOS=darwin GOARCH=amd64 go build -o bin/zeken-darwin cmd/zeken/main.go
	GOOS=windows GOARCH=amd64 go build -o bin/zeken.exe cmd/zeken/main.go
	chmod +x bin/*

build:
	rm -rf bin
	mkdir bin
	go build -o bin/zeken cmd/zeken/main.go
	@if [ "$(shell go env GOOS)" = "windows" ]; then \
		mv bin/zeken bin/zeken.exe; \
	fi
	chmod +x bin/*

.PHONY: clean
clean:
	rm -rf bin
