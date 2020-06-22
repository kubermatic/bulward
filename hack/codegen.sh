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

CRD_VERSION="v1"

# Manager
# -------
# CRDs
# The `|| true` is because the controller-gen will error out if CRD_types.go embeds CustomResourcDefinition, and it will be handled in the following yq removements.
$CONTROLLER_GEN crd:crdVersions=${CRD_VERSION} paths="./pkg/apis/core/..." output:crd:artifacts:config=config/internal/manager/crd/bases || true
# Webhooks
$CONTROLLER_GEN webhook paths="./pkg/manager/internal/webhooks/..." output:webhook:artifacts:config=config/internal/manager/webhook
# RBAC
$CONTROLLER_GEN rbac:roleName=manager-role paths="./pkg/manager/..." output:rbac:artifacts:config=config/internal/manager/rbac