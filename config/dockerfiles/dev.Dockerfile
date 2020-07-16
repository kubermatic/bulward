# Copyright 2020 The Bulward Authors.
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

# Development Tooling Container
# build by running `make build-image-dev`

FROM golang:1.14.2

RUN apt-get -qq update && \
  apt-get -qqy install ed curl gettext zip python3 python3-pip git jq make && \
  rm -rf /var/lib/apt/lists/* && \
  pip3 install pre-commit yq


# Allowed to use path@version
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV PATH=${PATH}:/usr/local/go/bin:${GOPATH}/bin
# this GOROOT ENV is needed for apiregister-gen.
ENV GOROOT=/usr/local/go

# versions without the `v` prefix
ARG APISERVER_BUILDER_VERSION
ARG CONTROLLER_GEN_VERSION

RUN echo $PATH && go get golang.org/x/tools/cmd/goimports && \
  go get sigs.k8s.io/controller-tools/cmd/controller-gen@v${CONTROLLER_GEN_VERSION} && \
  curl -sL https://github.com/kubernetes-sigs/apiserver-builder-alpha/releases/download/v${APISERVER_BUILDER_VERSION}/apiserver-builder-alpha-v${APISERVER_BUILDER_VERSION}-linux-amd64.tar.gz | tar -xz -C /usr/local
