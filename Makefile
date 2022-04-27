# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# IMAGE_VERSION represents the operator-manager, proxy-init versions.
# This value must be updated to the release tag of the most recent release, a change that must
# occur in the release commit.
export PROJECT_NAME = traffic-guru
export IMAGE_VERSION = v0.1.1-alpha2
# Build-time variables to inject into binaries
export SIMPLE_VERSION = $(shell (test "$(shell git describe --tags)" = "$(shell git describe --abbrev=0 --tags)" && echo $(shell git describe --tags)) || echo $(shell git describe --abbrev=0 --tags)+git)
export GIT_VERSION = $(shell git describe --dirty --tags --always)
export GIT_COMMIT = $(shell git rev-parse HEAD)
export BUILD_DATE ?= $(shell date +%Y-%m-%d-%H:%M-%Z)
export K8S_VERSION = 1.22.8
export CERT_MANAGER_VERSION = v1.7.2

export DEV_ARTIFACT_YAML = artifacts/$(PROJECT_NAME)-dev.yaml
export RELEASE_ARTIFACT_YAML = artifacts/$(PROJECT_NAME).yaml

# Build settings
export TOOLS_DIR = bin
#export SCRIPTS_DIR = tools/scripts
REPO = $(shell go list -m)
BUILD_DIR = bin

GO_ASMFLAGS ?= "all=-trimpath=$(shell dirname $(PWD))"
GO_ASMFLAGS_DEV ?= "all=-S"

GO_GCFLAGS ?= "all=-trimpath=$(shell dirname $(PWD))"
GO_GCFLAGS_DEV ?= "all=-N -l"

LDFLAGS_COMMON =  \
	-X $(REPO)/pkg/version.Version=$(SIMPLE_VERSION) \
	-X $(REPO)/pkg/version.GitVersion=$(GIT_VERSION) \
	-X $(REPO)/pkg/version.GitCommit=$(GIT_COMMIT) \
	-X $(REPO)/pkg/version.KubernetesVersion=v$(K8S_VERSION) \
	-X $(REPO)/pkg/version.ImageVersion=$(IMAGE_VERSION) \
	-X $(REPO)/pkg/version.BuildDate=$(BUILD_DATE)

GO_LDFLAGS ?= "$(LDFLAGS_COMMON) -s -w"
GO_LDFLAGS_DEV ?= "$(LDFLAGS_COMMON)"

GO_BUILD_ARGS = -gcflags $(GO_GCFLAGS) -asmflags $(GO_ASMFLAGS) -ldflags $(GO_LDFLAGS)
#GO_BUILD_ARGS_DEV = -gcflags $(GO_GCFLAGS_DEV) -asmflags $(GO_ASMFLAGS_DEV) -ldflags $(GO_LDFLAGS_DEV) -x
GO_BUILD_ARGS_DEV = -gcflags $(GO_GCFLAGS_DEV) -ldflags $(GO_LDFLAGS_DEV) -x

export GO111MODULE = on
export CGO_ENABLED = 0
export GOPROXY=https://goproxy.io
export PATH := $(PWD)/$(BUILD_DIR):$(PWD)/$(TOOLS_DIR):$(PATH)

export BUILD_IMAGE_REPO = flomesh
export IMAGE_TARGET_LIST = operator-manager proxy-init cluster-connector bootstrap ingress-pipy
IMAGE_PLATFORM = linux/amd64
ifeq ($(shell uname -m),arm64)
	IMAGE_PLATFORM = linux/arm64
endif

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
#CRD_OPTIONS ?= "crd:trivialVersions=false,preserveUnknownFields=false"
CRD_OPTIONS ?= "crd:generateEmbeddedObjectMeta=true"
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.22

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=$(PROJECT_NAME)-role crd paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out


##@ Build

.PHONY: build
build: generate fmt vet ## Build operator-manager, cluster-connector with release args, the result will be optimized.
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_ARGS) -o $(BUILD_DIR)/flomesh ./cli
	go build $(GO_BUILD_ARGS) -o $(BUILD_DIR) ./cmd/{operator-manager,cluster-connector,proxy-init,bootstrap,ingress-pipy}

.PHONY: build-dev
build-dev: generate fmt vet ## Build operator-manager, cluster-connector with debug args.
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_ARGS_DEV) -o $(BUILD_DIR)/flomesh ./cli
	go build $(GO_BUILD_ARGS_DEV) -o $(BUILD_DIR) ./cmd/{operator-manager,cluster-connector,proxy-init,bootstrap,ingress-pipy}

.PHONY: build/operator-manager build/cluster-connector build/proxy-init build/bootstrap build/ingress-pipy
build/operator-manager build/cluster-connector build/proxy-init build/bootstrap build/ingress-pipy:
	go build $(GO_BUILD_ARGS) -o $(BUILD_DIR)/$(@F) ./cmd/$(@F)

.PHONY: build/dev/operator-manager build/dev/cluster-connector build/dev/proxy-init build/dev/bootstrap build/dev/ingress-pipy
build/dev/operator-manager build/dev/cluster-connector build/dev/proxy-init build/dev/bootstrap build/dev/ingress-pipy:
	go build $(GO_BUILD_ARGS_DEV) -o $(BUILD_DIR)/$(@F) ./cmd/$(@F)

##@ Development

.PHONY: codegen
codegen: ## Generate ClientSet, Informer, Lister and Deepcopy code for Flomesh CRD
	./hack/update-codegen.sh

.PHONY: package-scripts
package-scripts: ## Tar all repo initializing scripts
	tar -C config/bootstrap/ -zcvf config/bootstrap/scripts.tar.gz scripts/

.PHONY: dev
#dev:  manifests build test kustomize ## Create dev commit changes to commit & Write dev commit changes.
dev:  manifests build-dev kustomize ## Create dev commit changes to commit & Write dev commit changes.
	export TRAFFIC_GURU_VERSION=$(IMAGE_VERSION)-dev && $(KUSTOMIZE) build config/overlays/dev/ | envsubst > $(DEV_ARTIFACT_YAML)
	sed -i '' 's/--v=2/--v=5/g' $(DEV_ARTIFACT_YAML)
	sed -i '' 's/proxy-init:latest/$(PROJECT_NAME)-proxy-init:dev/g' $(DEV_ARTIFACT_YAML)
	sed -i '' 's/cluster-connector:latest/$(PROJECT_NAME)-cluster-connector:dev/g' $(DEV_ARTIFACT_YAML)

.PHONY: build_docker_setup
build_docker_setup:
	docker run --rm --privileged tonistiigi/binfmt:latest --install all
	docker buildx create --name $(PROJECT_NAME)
	docker buildx use $(PROJECT_NAME)
	docker buildx inspect --bootstrap

.PHONY: rm_docker_builder
rm_docker_builder:
	docker buildx rm $(PROJECT_NAME) || true

.PHONY: build_docker_push
build_docker_push: build_docker_setup build_push_images rm_docker_builder

.PHONY: build_push_images
build_push_images: $(foreach i,$(IMAGE_TARGET_LIST),build_push_image/$(i))

build_push_image/%:
	docker buildx build --platform $(IMAGE_PLATFORM) \
		-t $(BUILD_IMAGE_REPO)/$(PROJECT_NAME)-$*:dev \
		-f ./dockerfiles/$*/Dockerfile \
		--push \
		--cache-from "type=local,src=.buildcache" \
		--cache-to "type=local,dest=.buildcache" \
		.

.PHONY: go-lint
go-lint:
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v1.45.2-alpine golangci-lint run --config .golangci.yml


##@ Release

.PHONY: check_release_version
check_release_version:
ifeq (,$(RELEASE_VERSION))
	$(error "RELEASE_VERSION must be set to a release tag")
endif
ifneq ($(RELEASE_VERSION),$(IMAGE_VERSION))
	$(error "IMAGE_VERSION "$(IMAGE_VERSION)" must be updated to match RELEASE_VERSION "$(RELEASE_VERSION)" prior to creating a release commit")
endif

.PHONY: gh-release
gh-release: ## Using goreleaser to Release target on Github.
ifeq (,$(GIT_VERSION))
	$(error "GIT_VERSION must be set to a git tag")
endif
	go install github.com/goreleaser/goreleaser@v1.6.3
	GORELEASER_CURRENT_TAG=$(GIT_VERSION) goreleaser release --rm-dist --parallelism 5

.PHONY: gh-release-snapshot
gh-release-snapshot:
ifeq (,$(GIT_VERSION))
	$(error "GIT_VERSION must be set to a git tag")
endif
	GORELEASER_CURRENT_TAG=$(GIT_VERSION) goreleaser release --snapshot --rm-dist --parallelism 5 --debug

.PHONY: gh-build-snapshot
gh-build-snapshot:
ifeq (,$(GIT_VERSION))
	$(error "GIT_VERSION must be set to a git tag")
endif
	GORELEASER_CURRENT_TAG=$(GIT_VERSION) goreleaser build --snapshot --rm-dist --parallelism 5 --debug


.PHONY: pre-release
pre-release: check_release_version manifests generate fmt vet kustomize edit_images  ## Create release commit changes to commit & Write release commit changes.
	export TRAFFIC_GURU_VERSION=$(RELEASE_VERSION) && $(KUSTOMIZE) build config/overlays/release/ | envsubst > $(RELEASE_ARTIFACT_YAML)
	echo "Replacing image tag to $(subst v,,$(IMAGE_VERSION))"
	sed -i '' 's/proxy-init:latest/$(PROJECT_NAME)-proxy-init:$(subst v,,$(IMAGE_VERSION))/g' $(RELEASE_ARTIFACT_YAML)
	sed -i '' 's/cluster-connector:latest/$(PROJECT_NAME)-cluster-connector:$(subst v,,$(IMAGE_VERSION))/g' $(RELEASE_ARTIFACT_YAML)

.PHONY: edit_images
edit_images: $(foreach i,$(IMAGE_TARGET_LIST),edit_image/$(i))

edit_image/%:
	cd config/overlays/release/ && $(KUSTOMIZE) edit set image $*=$(BUILD_IMAGE_REPO)/$(PROJECT_NAME)-$*:$(subst v,,$(IMAGE_VERSION))


.PHONY: release
VERSION_REGEXP := ^v[0-9]+\.[0-9]+\.[0-9]+(\-(alpha|beta|rc)\.[0-9]+)?$
release: ## Create a release tag, push to git repository and trigger the release workflow.
ifeq (,$(RELEASE_VERSION))
	$(error "RELEASE_VERSION must be set to tag HEAD")
endif
ifeq (,$(shell [[ "$(RELEASE_VERSION)" =~ $(VERSION_REGEXP) ]] && echo 1))
	$(error "Version $(RELEASE_VERSION) must match regexp $(VERSION_REGEXP)")
endif
	git tag --sign --message "$(PROJECT_NAME) $(RELEASE_VERSION)" $(RELEASE_VERSION)
	git verify-tag --verbose $(RELEASE_VERSION)
	git push origin --tags

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.2)

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

.DEFAULT_GOAL := help
.PHONY: help
help: ## Show this help screen.
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
