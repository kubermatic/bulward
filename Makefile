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

COMPONENTS = manager
IMAGE_ORG = quay.io/kubermatic
VERSION = v1
KIND_CLUSTER ?= bulward

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
generate:
	@hack/codegen.sh

deploy: generate kind-load-manager
	cd config/manager/manager && kustomize edit set image manager=${IMAGE_ORG}/bulward-manager:${VERSION}
	kustomize build config/manager/default | kubectl apply -f -

# ------------
# Test Runners
# ------------
test:
	go test -race -v ./...

lint: generate pre-commit
	@hack/validate-directory-clean.sh
	golangci-lint run ./... --deadline=15m

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
	@docker ps > /dev/null 2>&1 || start-docker.sh || (echo "cannot find running docker daemon nor can start new one" && false)
	@[[ -z "${QUAY_IO_USERNAME}" ]] || ( echo "logging in to ${QUAY_IO_USERNAME}" && docker login -u ${QUAY_IO_USERNAME} -p ${QUAY_IO_PASSWORD} quay.io )
.PHONY: require-docker

# ----------------
# Container Images
# ----------------
.SECONDEXPANSION:
build-image-%: bin/linux_amd64/$$* require-docker
	@mkdir -p bin/image/$*
	@mv bin/linux_amd64/$* bin/image/$*
	@cp -a config/dockerfiles/$*.Dockerfile bin/image/$*/Dockerfile
	@docker build -t ${IMAGE_ORG}/bulward-$*:${VERSION} bin/image/$*

kind-load-%: build-image-$$*
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
.PHONY: clean
