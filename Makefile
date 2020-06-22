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
IMAGE_ORG = quay.io/kubermatic/bulward
VERSION = v1
KIND_CLUSTER ?= bulward

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

deploy: kind-load-manager
	cd config/manager/manager && kustomize edit set image manager=${IMAGE_ORG}-manager:${VERSION}
	kustomize build config/manager/default | kubectl apply -f -

# ------------
# Test Runners
# ------------
test:
	go test -race -v ./...

lint: pre-commit
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
	@docker build -t ${IMAGE_ORG}-$*:${VERSION} bin/image/$*

kind-load-%: build-image-$$*
	kind load docker-image ${IMAGE_ORG}-$*:${VERSION} --name=${KIND_CLUSTER}

# -------
# Cleanup
# -------
clean:
	@rm -rf bin/$*
.PHONY: clean
