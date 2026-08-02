export PATH="/usr/local/go/bin:$PATH"
export GOTOOLCHAIN=local
go build -o main .
exec ./main server -d -p "${GOFLY_PORT:-8081}"
