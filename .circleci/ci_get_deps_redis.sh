#!/bin/sh
set -x
[ -f /tmp/redis-server ] && exit 0

wget http://download.redis.io/releases/redis-5.0.7.tar.gz
tar xvzf redis-5.0.7.tar.gz
cd redis-5.0.7 && make
cp src/redis-server /tmp
rm -rf redis-5.0.7*