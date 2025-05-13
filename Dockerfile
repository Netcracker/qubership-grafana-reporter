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

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY shutdown.go shutdown.go
COPY dashboard/ dashboard/
COPY handle/ handle/
COPY report/ report/
COPY timerange/ timerange/

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -a -o grafana-reporter .

# Final image
FROM ubuntu:24.04

ENV USER_UID=2001 \
    USER_NAME=appuser \
    GROUP_NAME=appuser \
    TINYTEX_URL="https://github.com/rstudio/tinytex-releases/releases/download/v2024.12/TinyTeX-0-v2024.12.tar.gz" \
    TEXDIR=/tinytex \
    BINDIR="$HOME/bin" \
    TLMGRDIR=/tinytex/.TinyTeX/bin/x86_64-linux

ENV PATH="$PATH:$TEXDIR"

WORKDIR /

COPY --from=builder --chown=${USER_UID} /workspace/grafana-reporter /bin/grafana-reporter
COPY templates/ /templates/

RUN apt-get -y update \
    && apt-get -f install -y \
        wget \
        perl \
    && apt-get clean \
    # Create TinyTex directory
    && mkdir -p /tinytex/ \
    && chmod -R +rwx /tinytex/ \
    # Download TinyTex
    && mkdir -p /tmp/tinytex/ \
    && wget --retry-connrefused --progress=dot:giga -O /tmp/tinytex/tinytexTinyTeX.tar.gz ${TINYTEX_URL} \
    && tar xzf /tmp/tinytex/tinytexTinyTeX.tar.gz -C ${TEXDIR} \
    && rm -rf /tmp/tinytex/tinytexTinyTeX.tar.gz \
    # Installation by tlmgr
    && perl ${TLMGRDIR}/tlmgr option sys_bin ${BINDIR} \
    && perl ${TLMGRDIR}/tlmgr postaction install script xetex \
    && perl ${TLMGRDIR}/tlmgr path add \
    # Create directories
    && mkdir -p /templates/custom/ /grafana/certificates/ /grafana/auth/ \
    # Grant permissions
    && chmod +x /bin/grafana-reporter \
    && chmod +rw /templates/ /grafana/ \
    && chown -R ${USER_UID}:${USER_UID} /templates/ /grafana/ /tinytex/

USER ${USER_UID}

ENTRYPOINT [ "/bin/grafana-reporter" ]
