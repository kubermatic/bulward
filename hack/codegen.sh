#!/usr/bin/env bash

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

set -euo pipefail

if [ -z $(go env GOBIN) ]; then
GOBIN=$(go env GOPATH)/bin
else
GOBIN=$(go env GOBIN)
fi

if [ -z $(which controller-gen) ]; then
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.9
  CONTROLLER_GEN=$GOBIN/controller-gen
else
  CONTROLLER_GEN=$(which controller-gen)
fi

CONTROLLER_GEN_VERSION=$(${CONTROLLER_GEN} --version)
CONTROLLER_GEN_WANT_VERSION="Version: v0.2.9"

if [[  ${CONTROLLER_GEN_VERSION} != ${CONTROLLER_GEN_WANT_VERSION} ]]; then
  echo "Wrong controller-gen version. Wants ${CONTROLLER_GEN_WANT_VERSION} found ${CONTROLLER_GEN_VERSION}"
  exit 1
fi

APISERVER_BOOT=$(which apiserver-boot)

# DeepCopy functions
$CONTROLLER_GEN object:headerFile=./hack/boilerplate/boilerplate.go.txt,year=$(date +%Y) paths=./pkg/apis/...

CRD_VERSION="v1"

# Manager
# -------
# CRDs
$CONTROLLER_GEN crd:crdVersions=${CRD_VERSION} paths="./pkg/apis/core/..." output:crd:artifacts:config=config/manager/crd/bases
# Webhooks
$CONTROLLER_GEN webhook paths="./pkg/manager/internal/webhooks/..." output:webhook:artifacts:config=config/manager/webhook
# RBAC
$CONTROLLER_GEN rbac:roleName=manager-role paths="./pkg/manager/..." output:rbac:artifacts:config=config/manager/rbac


# Bulward API extension server
# RBAC
$CONTROLLER_GEN rbac:roleName=manager-role paths="./pkg/apiserver/..." output:rbac:artifacts:config=config/apiserver/rbac
# Generators for API extension server.
$APISERVER_BOOT build generated  --generator apiregister --generator conversion  --generator openapi --generator defaulter
#find ./pkg -type f -name '*.go' -exec sed -i'' 's/YEAR/2020/g' {} \;
goimports -local github.com/kubermatic -w .
