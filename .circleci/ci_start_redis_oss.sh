#!/bin/sh
OSS_REDIS_PATH=${OSS_REDIS_PATH:-"/tmp/redis-server"}
SHARD_COUNT=${SHARD_COUNT:-5}
MODULE=${MODULE:-"/tmp/redistimeseries.so"}
PIDSAVE=${PIDSAVE:-"/tmp/rltestpid"}

set -x

python2.7 -m RLTest --env oss --env-only \
          --oss-redis-path ${OSS_REDIS_PATH} \
          --module ${MODULE}  > /dev/null 2>&1 &
echo $! >> ${PIDSAVE}
