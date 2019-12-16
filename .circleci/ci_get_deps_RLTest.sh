#!/bin/sh
set -x
python2.7 -m pip install wheel

python2.7 -m pip install setuptools --upgrade
python2.7 -m pip install git+https://github.com/RedisLabsModules/RLTest.git@master
python2.7 -m pip install redis-py-cluster
