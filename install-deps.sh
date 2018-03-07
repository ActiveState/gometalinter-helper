#!/bin/bash

set -eo pipefail

go get -u gopkg.in/alecthomas/gometalinter.v2
# includes aligncheck, structcheck, & varcheck
go get -u github.com/opennota/check
go get -u github.com/kisielk/errcheck
go get -u github.com/golang/lint/golint
go get -u honnef.co/go/tools/cmd/megacheck
go get -u github.com/client9/misspell/cmd/misspell

