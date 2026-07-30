#!/bin/sh
set -eu

if [ ! -x ./main ]; then
	echo "main binary does not exist; start the service first" >&2
	exit 1
fi

exec ./main stop
