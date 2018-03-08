#!/bin/bash

set -eo pipefail

go get -u gopkg.in/alecthomas/gometalinter.v2

go get -u github.com/client9/misspell/cmd/misspell
go get -u github.com/golang/lint/golint
go get -u github.com/kisielk/errcheck
go get -u github.com/opennota/check/cmd/structcheck
go get -u github.com/tsenart/deadcode
go get -u honnef.co/go/tools/cmd/megacheck
