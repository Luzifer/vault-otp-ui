default: build

ci:
	curl -sSLo golang.sh https://raw.githubusercontent.com/Luzifer/github-publish/master/golang.sh
	bash golang.sh

build: generate
	go build -ldflags "-X main.version=$(shell git describe --tags || git rev-parse --short HEAD || echo dev)"

install: generate
	go install -ldflags "-X main.version=$(shell git describe --tags || git rev-parse --short HEAD || echo dev)"

generate:
	go generate
