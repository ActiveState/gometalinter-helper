#!/bin/bash

set -eo pipefail

command="go run ./cmd/gometalinter-helper/main.go"
if [ "$@" ]; then
    command=$@
fi
$command \
    -verbose \
    -exe gometalinter.v2 \
    -all \
    -- \
    --vendor \
    --disable-all \
    --sort=path \
    --enable=deadcode \
    --enable=errcheck \
    --enable=gofmt \
    --enable=golint \
    --enable=megacheck \
    --enable=misspell \
    --enable=structcheck \
