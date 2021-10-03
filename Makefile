TARGETS=darwin linux windows

default: build

build:
	go build -v

test:
	go test -v ./...

dockerbuild:
	docker run --rm -it -v `pwd`:/workspace -w /workspace golang:1.17 go build -v

dockertest:
	docker run --rm -it -v `pwd`:/workspace -w /workspace golang:1.17 go test -v ./...

release:
	docker run --rm -it -v `pwd`:/workspace -w /workspace -e GITHUB_TOKEN golang:1.17 /workspace/scripts/release.sh

snapshot:
	docker run --rm -it -v `pwd`:/workspace -w /workspace -e GITHUB_TOKEN golang:1.17 /workspace/scripts/release.sh --snapshot

.PHONY: build test changelog targets $(TARGETS)
