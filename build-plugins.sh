#!/bin/bash
#
# Copyright 2023-2024 EMQ Technologies Co., Ltd.
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

set -euo pipefail

cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")"

PLUGIN_TYPE=$1
PLUGIN_NAME=$2
VERSION=$(git describe --tags --always --match 'v[0-9]*.[0-9]*.[0-9]*' | sed 's/^v//g')
OS=$(sed -n '/^ID=/p' /etc/os-release | sed -r 's/ID=(.*)/\1/g')

pre(){
    mkdir -p _plugins/$OS/$PLUGIN_TYPE
}

# 插件是 buildmode=plugin, 即动态链接(external linking)构建。Go 1.27 的汇编器在
# dynlink 下会拒绝"跨全局符号访问使用 R15"的汇编, 而 pierrec/lz4/v4 (经 trino 驱动
# 被 sql sink 与 kafka 插件带入) 的 decode_amd64.s 正是这种写法, 于是插件构建直接
# 失败在 "asm: assembly failed"。该写法在 4.1.30~4.1.34 之间没有变化, 靠升依赖解决
# 不了, 而 4.1.23~4.1.29 又被上游标记为 retracted(有丢数据缺陷), 不能降。
# 因此插件构建统一带 noasm: lz4 与 xxh32 回退到纯 Go 解码器, 只是慢一些, 语义不变。
PLUGIN_TAGS="noasm"

post(){
    if [ -f "etc/$PLUGIN_TYPE/$PLUGIN_NAME.yaml" ]; then
        cp etc/$PLUGIN_TYPE/$PLUGIN_NAME.yaml extensions/$PLUGIN_TYPE/$PLUGIN_NAME;
    fi
    cd extensions/$PLUGIN_TYPE/$PLUGIN_NAME
    zip -r ${PLUGIN_NAME}_$(go env GOARCH).zip .
    cd -
    mv $(find extensions/$PLUGIN_TYPE/$PLUGIN_NAME -name "*.zip")  _plugins/$OS/$PLUGIN_TYPE
}

build(){
    case $PLUGIN_NAME in
        influx )
            go build -trimpath --buildmode=plugin -tags "plugins,$PLUGIN_TAGS" -o extensions/$PLUGIN_TYPE/$PLUGIN_NAME/$PLUGIN_NAME@$VERSION.so extensions/$PLUGIN_TYPE/$PLUGIN_NAME/$PLUGIN_NAME.go
            ;;
        tdengine )
            if [ "$(uname -m)" = "x86_64" ]; then
                wget "https://www.taosdata.com/assets-download/TDengine-client-2.4.0.18-Linux-x64.tar.gz" -O /tmp/TDengine-client-2.4.0.18.tar.gz;
            fi;
            if [ "$(uname -m)" = "aarch64" ]; then
                wget "https://www.taosdata.com/assets-download/TDengine-client-2.4.0.18-Linux-aarch64.tar.gz" -O /tmp/TDengine-client-2.4.0.18.tar.gz;
            fi;
            tar -zxvf /tmp/TDengine-client-2.4.0.18.tar.gz
            cd TDengine-client-2.4.0.18 && ./install_client.sh && cd -
            go build -trimpath --buildmode=plugin -tags "plugins,$PLUGIN_TAGS" -o extensions/$PLUGIN_TYPE/$PLUGIN_NAME/$PLUGIN_NAME@$VERSION.so extensions/$PLUGIN_TYPE/$PLUGIN_NAME/$PLUGIN_NAME.go
            ;;
        labelImage )
            if [ ! -d "/tmp/tensorflow" ];then
                git clone -b v2.18.1 --depth 1 https://github.com/tensorflow/tensorflow.git /tmp/tensorflow;
            fi;
            if [ "$(uname -m)" = "x86_64" ]; then
                cp $(pwd)/extensions/functions/dependencies/tensorflow/amd64/*.so $(pwd)/extensions/functions/labelImage/lib
            fi;
            if [ "$(uname -m)" = "aarch64" ]; then
                cp $(pwd)/extensions/functions/dependencies/tensorflow/arm64/*.so $(pwd)/extensions/functions/labelImage/lib
            fi;
            CGO_CFLAGS=-I/tmp/tensorflow CGO_LDFLAGS=-L$(pwd)/extensions/functions/labelImage/lib go build -trimpath --buildmode=plugin -tags "$PLUGIN_TAGS" -o extensions/functions/labelImage/labelImage@$VERSION.so extensions/functions/labelImage/*.go
            ;;
        tfLite )
            if [ ! -d "/tmp/tensorflow" ];then
                git clone -b v2.18.1 --depth 1 https://github.com/tensorflow/tensorflow.git /tmp/tensorflow;
            fi;
            if [ "$(uname -m)" = "x86_64" ]; then
                cp $(pwd)/extensions/functions/dependencies/tensorflow/amd64/*.so $(pwd)/extensions/functions/tfLite/lib
            fi;
            if [ "$(uname -m)" = "aarch64" ]; then
                cp $(pwd)/extensions/functions/dependencies/tensorflow/arm64/*.so $(pwd)/extensions/functions/tfLite/lib
            fi;
                CGO_CFLAGS=-I/tmp/tensorflow CGO_LDFLAGS=-L$(pwd)/extensions/functions/tfLite/lib go build -trimpath -tags "tflite,$PLUGIN_TAGS" --buildmode=plugin -o extensions/functions/tfLite/tfLite@$VERSION.so extensions/functions/tfLite/*.go
            ;;
        * )
            go build -trimpath --buildmode=plugin -tags "$PLUGIN_TAGS" -o extensions/$PLUGIN_TYPE/$PLUGIN_NAME/$PLUGIN_NAME@$VERSION.so extensions/$PLUGIN_TYPE/$PLUGIN_NAME/*.go
            ;;
    esac
}

pre
build
post
