# Copyright 2024-2025 NetCracker Technology Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# hadolint global ignore=DL3008
FROM --platform=$BUILDPLATFORM golang:1.24.2-alpine3.21 AS builder
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY main.go shutdown.go ./
COPY dashboard/ dashboard/
COPY handle/ handle/
COPY report/ report/
COPY timerange/ timerange/

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -a -o grafana-reporter .

# Final image
FROM ubuntu:24.04

ARG TARGETARCH
ENV USER_UID=2001 \
    USER_NAME=appuser \
    GROUP_NAME=appuser \
    TINYTEX_URL="https://github.com/rstudio/tinytex-releases/releases/download/v2024.12/TinyTeX-0-v2024.12.tar.gz" \
    TEXDIR=/tinytex \
    BINDIR="$HOME/bin"

WORKDIR /

COPY --from=builder --chown=${USER_UID} /workspace/grafana-reporter /bin/grafana-reporter
COPY templates/ /templates/

RUN apt-get -y update \
    && apt-get install -y wget perl \
    && apt-get clean \
    && mkdir -p /templates/custom/ /grafana/certificates/ /grafana/auth/ \
    && chmod +x /bin/grafana-reporter \
    && chmod +rw /templates/ /grafana/ \
    && chown -R ${USER_UID}:${USER_UID} /templates/ /grafana/

# Only install TinyTeX if on amd64
RUN if [ "$TARGETARCH" = "amd64" ]; then \
      mkdir -p /tinytex/ && chmod -R +rwx /tinytex/ \
      && mkdir -p /tmp/tinytex/ \
      && wget --retry-connrefused --progress=dot:giga -O /tmp/tinytex/tinytexTinyTeX.tar.gz ${TINYTEX_URL} \
      && tar xzf /tmp/tinytex/tinytexTinyTeX.tar.gz -C ${TEXDIR} \
      && rm -rf /tmp/tinytex/tinytexTinyTeX.tar.gz \
      && perl /tinytex/.TinyTeX/bin/x86_64-linux/tlmgr option sys_bin ${BINDIR} \
      && perl /tinytex/.TinyTeX/bin/x86_64-linux/tlmgr postaction install script xetex \
      && perl /tinytex/.TinyTeX/bin/x86_64-linux/tlmgr path add \
      && chown -R ${USER_UID}:${USER_UID} /tinytex/ ; \
    fi

USER ${USER_UID}
ENTRYPOINT [ "/bin/grafana-reporter" ]

