#!/bin/bash
set -eo pipefail
cd "$(dirname "$0")"
export PATH="/usr/local/go/bin:$PATH"
go test ./pkg/charutil/... ./pkg/scoring/... ./pkg/ansi/... ./pkg/sync/... ./pkg/util/... ./pkg/tokenizer/... ./pkg/chunk/... ./pkg/result/... -count=1
