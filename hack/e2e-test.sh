#!/usr/bin/env bash

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

# This file should ONLY be called from within Makefile!!!
set -euo pipefail

JOB_LOG=${PULL_NUMBER:-}-${JOB_NAME:-}-${BUILD_ID:-}
workdir=$(mktemp -d)
if [[ "${JOB_LOG}" != "--" ]]; then
  workdir=${workdir}/${JOB_LOG}
  mkdir -p ${workdir}
fi

function cleanup() {
  echo "starting cleanup & log upload"
  kind export logs --name bulward ${workdir}
  docker cp bulward-control-plane:/var/log/kube-apiserver-audit.log ${workdir}/audit.log
  echo "find all logs in ${workdir}"

  # https://github.com/kubernetes/test-infra/blob/master/prow/jobs.md#job-environment-variables
  if [[ "${JOB_LOG}" != "--" ]]; then
    tmpdir=$(mktemp -d)
    pushd ${workdir};
    zip --quiet -r "${tmpdir}/${JOB_LOG}.zip" .
    popd;

    # TODO Implement bulward one
    aws s3 cp "${tmpdir}/${JOB_LOG}.zip" "s3://e2elogs.kubecarrier.io/${JOB_LOG}.zip"
    echo "https://s3.eu-central-1.amazonaws.com/e2elogs.kubecarrier.io/${JOB_LOG}.zip"
  fi
}

trap cleanup EXIT
go test ./test/... | tee ${workdir}/test.out
