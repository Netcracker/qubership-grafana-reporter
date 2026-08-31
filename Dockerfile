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
FROM golang:1.27.0-alpine3.24@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS builder

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
COPY utils/ utils/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o grafana-reporter .

# Final image
FROM ubuntu:26.04@sha256:678c6550cc43645e08669028bc177f50be4e7c5b8cca677067b1914d4afc7a03

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
