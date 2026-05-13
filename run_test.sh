#!/bin/bash
set -eo pipefail
cd "$(dirname "$0")"
export PATH="/usr/local/go/bin:$PATH"
go test ./pkg/charutil/... ./pkg/scoring/... ./pkg/ansi/... ./pkg/sync/... ./pkg/util/... ./pkg/tokenizer/... ./pkg/chunk/... ./pkg/result/... ./pkg/pattern/... ./pkg/merger/... ./pkg/history/... ./pkg/reader/... ./pkg/matcher/... -count=1
