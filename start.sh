#!/bin/sh
set -eu

go build -o main .
exec ./main server -d -p "${GOFLY_PORT:-8081}"
