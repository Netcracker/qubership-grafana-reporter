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

FROM golang:1.21.6-alpine3.18 AS builder

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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o grafana-reporter .

FROM ubuntu:24.04

RUN apt-get -y update \
    && apt-get -f install -y wget perl \
    && apt-get clean

WORKDIR /

RUN mkdir -p /tinytex/ \
    && chmod -R +rwx /tinytex/

ENV USER_UID=2001 \
    USER_NAME=appuser \
    GROUP_NAME=appuser \
    TINYTEX_URL="https://github.com/rstudio/tinytex-releases/releases/download/v2024.12/TinyTeX-0-v2024.12.tar.gz" \
    TEXDIR=/tinytex \
    BINDIR="$HOME/bin" \
    TLMGRDIR=/tinytex/.TinyTeX/bin/x86_64-linux

RUN addgroup ${GROUP_NAME}
RUN adduser --ingroup ${GROUP_NAME} -uid ${USER_UID} ${USER_NAME}

# download tinytex
RUN mkdir -p /tmp/tinytex/
RUN wget --retry-connrefused --progress=dot:giga -O /tmp/tinytex/tinytexTinyTeX.tar.gz ${TINYTEX_URL}
RUN tar xzf /tmp/tinytex/tinytexTinyTeX.tar.gz -C ${TEXDIR}
RUN rm /tmp/tinytex/tinytexTinyTeX.tar.gz

ENV PATH="$PATH:${TEXDIR}"

# installation by tlmgr
RUN perl ${TLMGRDIR}/tlmgr option sys_bin ${BINDIR}
RUN perl ${TLMGRDIR}/tlmgr postaction install script xetex
RUN perl ${TLMGRDIR}/tlmgr path add

COPY templates/ /templates/

RUN mkdir -p /templates/custom/ /grafana/certificates/ /grafana/auth/

COPY --from=builder --chown=${USER_UID} /workspace/grafana-reporter /bin/grafana-reporter

RUN chmod +x /bin/grafana-reporter
RUN chmod +rw /templates/ /grafana/
RUN chown -R ${USER_UID}:${USER_UID} /templates/ /grafana/ /tinytex/

USER ${USER_UID}

ENTRYPOINT [ "/bin/grafana-reporter" ]
