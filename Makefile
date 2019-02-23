
default: build

build: 
	go build -v

test:
	go test -v ./...

.PHONY: build test