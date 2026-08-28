# Copyright 2021-2023 EMQ Technologies Co., Ltd.
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

ARG GO_VERSION=1.25.4
FROM ghcr.io/lf-edge/ekuiper/base:$GO_VERSION-alpine AS builder

COPY . /go/kuiper

WORKDIR /go/kuiper

RUN make build_with_edgex

FROM alpine:3.20.0

# Set environment vars
ENV MAINTAINER="b2c" \
    KUIPER_HOME="/b2c" \
    KUIPER__BASIC__CONSOLELOG=true

# These vars are not persisted in the final image layer
ARG KUIPER_USER="b2c"
ARG KUIPER_USER_ID="1001"

WORKDIR ${KUIPER_HOME}

# Set appropriate ownership to allow binary full access to KUIPER_HOME dir
RUN adduser -DH -s /sbin/nologin -u ${KUIPER_USER_ID} ${KUIPER_USER} && \
    chown -Rh ${KUIPER_USER}:${KUIPER_USER} ${KUIPER_HOME} && \
    mkdir -p /usr/local/taos && \
    chown -Rh ${KUIPER_USER}:${KUIPER_USER} /usr/local/taos

# Run the b2c process under the b2c user
USER ${KUIPER_USER}

COPY --chown=${KUIPER_USER}:${KUIPER_USER} ./deploy/docker/docker-entrypoint.sh /usr/bin/docker-entrypoint.sh
COPY --chown=${KUIPER_USER}:${KUIPER_USER} --from=builder /go/kuiper/_build/b2c-* /b2c/

VOLUME ["${KUIPER_HOME}/etc", "${KUIPER_HOME}/data", "${KUIPER_HOME}/plugins", "${KUIPER_HOME}/log"]
EXPOSE 9081 20498

ENTRYPOINT ["/usr/bin/docker-entrypoint.sh"]

CMD ["./bin/b2cd"]
