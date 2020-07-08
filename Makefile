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

CERTMANAGER_VERSION = v0.14.0
COMPONENTS = manager apiserver
IMAGE_ORG = quay.io/kubermatic
KIND_CLUSTER ?= bulward
KUBECONFIG ?= ${HOME}/.kube/kind-config-${KIND_CLUSTER}
SHELL=/bin/bash
.SHELLFLAGS=-euo pipefail -c
VERSION = v1

export CGO_ENABLED:=0

ifdef CI
	# prow sets up GOPATH and we want to make sure it's in the PATH
	# https://github.com/kubernetes/test-infra/issues/9469
	# https://github.com/kubernetes/test-infra/blob/895df89b7e4238125063157842c191dac6f7e58f/prow/pod-utils/decorate/podspec.go#L474
	export PATH:=${PATH}:${GOPATH}/bin
endif


# -----------------
# Compile & Release
# -----------------
bin/linux_amd64/%: GOARGS = GOOS=linux GOARCH=amd64
bin/darwin_amd64/%: GOARGS = GOOS=darwin GOARCH=amd64
bin/windows_amd64/%: GOARGS = GOOS=windows GOARCH=amd64

bin/%:
	$(eval COMPONENT=$(shell basename $*))
	$(GOARGS) go build  -o bin/$* cmd/$(COMPONENT)/main.go

# ---------------
# Code Generators
# ---------------

# This should only be executed once the new crd is added in the apiserver api group.
generate-apiregister:
	apiregister-gen --input-dirs github.com/kubermatic/bulward/pkg/apis/apiserver/... --input-dirs github.com/kubermatic/bulward/pkg/apis --go-header-file ./hack/boilerplate/boilerplate.go.txt

generate:
	@hack/codegen.sh

setup-cluster: require-docker
	@mkdir -p /tmp/bulward-hack
	@cp ./hack/audit.yaml /tmp/bulward-hack
	@kind create cluster --retain --config=./hack/kind-config.yaml --name=${KIND_CLUSTER} --image=${KIND_NODE_IMAGE} || true
	@kind get kubeconfig --name=${KIND_CLUSTER} > "${KUBECONFIG}"
	@kubectl create namespace bulward-system || true  # ignore if exists

deploy-manager: setup-cluster kind-load-manager
	@echo "deploying manager"
	@kubectl apply -k config/manager/default -o yaml --dry-run | sed "s|quay.io/kubermatic/bulward-manager:v1|${IMAGE_ORG}/bulward-manager:${VERSION}|g" | kubectl apply -f -
	kubectl wait --for=condition=available deployment/bulward-controller-manager -n bulward-system --timeout=120s

deploy-apiserver: setup-cluster cert-manager kind-load-apiserver
	@echo "deploying API extension server"
	@#kubectl 1.18.3 has older kustomize version which improperly renders APIService name
	@kustomize build config/apiserver/default | sed "s|quay.io/kubermatic/bulward-manager:v1|${IMAGE_ORG}/bulward-manager:${VERSION}|g" | kubectl apply -f -
	@kubectl apply -f config/apiserver/rbac/extension_apiserver_auth_role_binding.yaml
	kubectl wait --for=condition=available deployment/bulward-apiserver-controller-manager -n bulward-system --timeout=120s

deploy: deploy-apiserver deploy-manager

e2e-test: deploy
	@./hack/e2e-test.sh

# ------------
# Test Runners
# ------------
test:
	CGO_ENABLED=1 go test -race -v ./pkg/...
.PHONY: test

lint: pre-commit
	@hack/validate-directory-clean.sh
	golangci-lint run ./... --skip-files ".*generated.*" --deadline=15m

# -------------
# Util Commands
# -------------
fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

pre-commit:
	pre-commit run -a

require-docker:
	@docker ps > /dev/null 2>&1 || start-docker.sh || ./hack/start-docker.sh || (echo "cannot find running docker daemon nor can start new one" && false)
	@[[ -z "${QUAY_IO_USERNAME}" ]] || ( echo "logging in to ${QUAY_IO_USERNAME}" && docker login -u ${QUAY_IO_USERNAME} -p ${QUAY_IO_PASSWORD} quay.io )
.PHONY: require-docker

# Install cert-manager in the configured Kubernetes cluster
cert-manager: setup-cluster
	@echo "Deploy cert-manger in management cluster"
	docker pull quay.io/jetstack/cert-manager-controller:${CERTMANAGER_VERSION}
	docker pull quay.io/jetstack/cert-manager-cainjector:${CERTMANAGER_VERSION}
	docker pull quay.io/jetstack/cert-manager-webhook:${CERTMANAGER_VERSION}
	kind load docker-image quay.io/jetstack/cert-manager-controller:${CERTMANAGER_VERSION} --name=${KIND_CLUSTER}
	kind load docker-image quay.io/jetstack/cert-manager-cainjector:${CERTMANAGER_VERSION} --name=${KIND_CLUSTER}
	kind load docker-image quay.io/jetstack/cert-manager-webhook:${CERTMANAGER_VERSION} --name=${KIND_CLUSTER}
	kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/${CERTMANAGER_VERSION}/cert-manager.yaml
	kubectl wait --for=condition=available deployment/cert-manager -n cert-manager --timeout=120s
	kubectl wait --for=condition=available deployment/cert-manager-cainjector -n cert-manager --timeout=120s
	kubectl wait --for=condition=available deployment/cert-manager-webhook -n cert-manager --timeout=120s

# ----------------
# Container Images
# ----------------
.SECONDEXPANSION:
build-image-%: bin/linux_amd64/$$* require-docker
	@mkdir -p bin/image/$*
	@mv bin/linux_amd64/$* bin/image/$*
	@cp -a config/dockerfiles/$*.Dockerfile bin/image/$*/Dockerfile
	@docker build -t ${IMAGE_ORG}/bulward-$*:${VERSION} bin/image/$*

kind-load-%: build-image-$$* setup-cluster
	kind load docker-image ${IMAGE_ORG}/bulward-$*:${VERSION} --name=${KIND_CLUSTER}

build-image-test: require-docker
	@mkdir -p bin/image/test
	@cp -a config/dockerfiles/test.Dockerfile bin/image/test/Dockerfile
	@cp -a .pre-commit-config.yaml bin/image/test
	@cp -a go.mod go.sum bin/image/test
	@cp -a hack/start-docker.sh bin/image/test
	@docker build -t ${IMAGE_ORG}/bulward-test bin/image/test

push-image-test: build-image-test require-docker
	@docker push ${IMAGE_ORG}/bulward-test
	@echo pushed ${IMAGE_ORG}/bulward-test

# -------
# Cleanup
# -------
clean:
	@rm -rf bin/$*
	@kind delete cluster --name=${KIND_CLUSTER}
.PHONY: clean
