# Copyright 2019 The Bulward Authors.
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

# Should be build from the Makefile, `make build-image-test`

FROM golang:1.14.2

RUN apt-get -qq update && apt-get -qqy install \
  curl \
  zip \
  python3-pip \
  git \
  make \
  && rm -rf /var/lib/apt/lists/*

ENV PATH=${PATH}:/usr/local/go/bin:/root/go/bin
# Allowed to use path@version
ENV GO111MODULE=on
RUN go env

RUN go get golang.org/x/tools/cmd/goimports
RUN pip3 install pre-commit
# binary will be $(go env GOPATH)/bin/golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.27.0

WORKDIR /src

# Create pre-commit cache, that is download required pre-commit repos
COPY .pre-commit-config.yaml .pre-commit-config.yaml
RUN git init && (pre-commit run || true) && rm -Rvf .git
