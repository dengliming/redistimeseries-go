#!/bin/sh
set -x
[ -f /tmp/redistimeseries.so ] && exit 0

git clone --recursive https://github.com/RedisTimeSeries/RedisTimeSeries.git
cd RedisTimeSeries
make setup
make build
cp bin/redistimeseries.so /tmp
rm -rf RedisTimeSeries

