TARGETS=darwin linux windows

default: build

build: 
	go build -v

test:
	go test -v ./...

release:
	docker run --rm -it -v `pwd`:/workspace -w /workspace -e GITHUB_TOKEN golang:1.13 /workspace/scripts/release.sh

snapshot:
	GOPROXY=direct docker run --rm -it -v `pwd`:/workspace -w /workspace -e GITHUB_TOKEN golang:1.13 /workspace/scripts/release.sh --snapshot

.PHONY: build test changelog targets $(TARGETS)