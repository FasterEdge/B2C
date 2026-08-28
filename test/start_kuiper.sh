#!/bin/bash
#
# Copyright 2021-2024 EMQ Technologies Co., Ltd.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -e

pids=`ps aux|grep "b2cd" | grep "bin"|awk '{printf $2 " "}'`
if [ "$pids" = "" ] ; then
   echo "No b2c server was started"
else
  for pid in $pids ; do
    echo "kill b2c " $pid
    kill -9 $pid
  done
fi

ver=`git describe --tags --always --match 'v[0-9]*.[0-9]*.[0-9]*' | sed 's/^v//g'`
os=`uname -s | tr "[A-Z]" "[a-z]"`
base_dir=_build/b2c-"$ver"-"$os"-amd64
rm -rf $base_dir/data/*
ls -l $base_dir/bin/b2cd

mkdir -p cover
export GOCOVERDIR="../../cover"
export BUILD_ID=dontKillMe
export KUIPER__BASIC__PROMETHEUS="true"
export KUIPER__BASIC__PROMETHEUSPORT=9081
export KUIPER__BASIC__RESTPORT=9081
export KUIPER__PORTABLE__INITTIMEOUT="5m"
export KUIPER__BASIC__ENABLEPRIVATENET="true"

cd $base_dir/
touch log/b2c.out
nohup bin/b2cd > log/b2c.out 2>&1 &
echo "starting b2c at " $base_dir