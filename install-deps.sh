#!/bin/bash

set -eo pipefail

go get -u github.com/golang/lint/golint
go get -u github.com/kisielk/errcheck
go get -u github.com/golang/dep/cmd/dep

