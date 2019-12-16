#!/bin/sh
PIDSAVE=${PIDSAVE:-"/tmp/rltestpid"}

set -x

kill -9 $(cat ${PIDSAVE})
kill -9 $(pgrep redis-server)
