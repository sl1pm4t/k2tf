#!/bin/bash

# Build and release
cd /workspace
curl -sL https://git.io/goreleaser | bash -s -- --rm-dist $@