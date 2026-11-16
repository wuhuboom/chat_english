#!/usr/bin/env sh
set -eu
cd "$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
if [ ! -f config.yaml ]; then cp config.yaml.example config.yaml; fi
if [ ! -f config/mysql.json ]; then
  cp config/mysql.json.example config/mysql.json
  chmod 600 config/mysql.json
  echo 'Edit config/mysql.json with your existing MySQL connection, then run again.'
  exit 1
fi
mkdir -p logs static/upload
if [ ! -f install.lock ]; then printf 'existing-database\n' > install.lock; fi
exec ./main server -p "${GOFLY_PORT:-8081}"
