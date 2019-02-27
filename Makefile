TARGETS=darwin linux windows

GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
GIT_SUMMARY ?= $(shell git describe --tags --dirty --always)
# VERSION ?= $(shell )

default: build

build: 
	go build -v

test:
	go test -v ./...

targets: $(TARGETS)

$(TARGETS):
	GOOS=$@ GOARCH=amd64 CGO_ENABLED=0 go build -v -o "dist/$@/k2tf" -a -ldflags '-extldflags "-static"'
	zip -j dist/k2tf_${TRAVIS_TAG}_$@_amd64.zip dist/$@/k2tf

changelog:
	github_changelog_generator --user sl1pm4t --project k2tf --release-branch master

.PHONY: build test changelog targets $(TARGETS)