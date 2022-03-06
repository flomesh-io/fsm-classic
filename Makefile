# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# IMAGE_VERSION represents the operator-manager, proxy-init versions.
# This value must be updated to the release tag of the most recent release, a change that must
# occur in the release commit.
export IMAGE_VERSION = v0.0.0
# Build-time variables to inject into binaries
export SIMPLE_VERSION = $(shell (test "$(shell git describe --tags)" = "$(shell git describe --abbrev=0 --tags)" && echo $(shell git describe --tags)) || echo $(shell git describe --abbrev=0 --tags)+git)
export GIT_VERSION = $(shell git describe --dirty --tags --always)
export GIT_COMMIT = $(shell git rev-parse HEAD)
export K8S_VERSION = 1.21.8
export CERT_MANAGER_VERSION = v1.5.3

export DEV_ARTIFCAT_YAML = artifact/flomesh-service-mesh-dev.yaml
export RELEASE_ARTIFCAT_YAML = artifact/flomesh-service-mesh.yaml

# Build settings
export TOOLS_DIR = bin
#export SCRIPTS_DIR = tools/scripts
REPO = $(shell go list -m)
BUILD_DIR = bin
GO_ASMFLAGS = -asmflags "all=-trimpath=$(shell dirname $(PWD))"
GO_GCFLAGS = -gcflags "all=-trimpath=$(shell dirname $(PWD))"
GO_BUILD_ARGS = \
  $(GO_GCFLAGS) $(GO_ASMFLAGS) \
  -ldflags " \
    -X '$(REPO)/pkg/version.Version=$(SIMPLE_VERSION)' \
    -X '$(REPO)/pkg/version.GitVersion=$(GIT_VERSION)' \
    -X '$(REPO)/pkg/version.GitCommit=$(GIT_COMMIT)' \
    -X '$(REPO)/pkg/version.KubernetesVersion=v$(K8S_VERSION)' \
    -X '$(REPO)/pkg/version.ImageVersion=$(IMAGE_VERSION)' \
  " \

export GO111MODULE = on
export CGO_ENABLED = 0
export GOPROXY=https://goproxy.io
export PATH := $(PWD)/$(BUILD_DIR):$(PWD)/$(TOOLS_DIR):$(PATH)

export BUILD_IMAGE_REPO = flomesh
export IMAGE_TARGET_LIST = operator-manager proxy-init cluster-connector repo-init ingress-pipy

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
#CRD_OPTIONS ?= "crd:trivialVersions=false,preserveUnknownFields=false"
CRD_OPTIONS ?= "crd:generateEmbeddedObjectMeta=true"
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.21

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=flomesh-service-mesh-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

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
build: generate fmt vet ## Build operator-manager cluster-connector.
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_ARGS) -o $(BUILD_DIR)/flomesh ./cli
	go build $(GO_BUILD_ARGS) -o $(BUILD_DIR) ./cmd/{operator-manager,cluster-connector}

.PHONY: build/operator-manager build/cluster-connector
build/operator-manager build/cluster-connector:
	go build $(GO_BUILD_ARGS) -o $(BUILD_DIR)/$(@F) ./cmd/$(@F)

##@ Development

.PHONY: dev
#dev:  manifests build test kustomize ## Create dev commit changes to commit & Write dev commit changes.
dev:  manifests build kustomize ## Create dev commit changes to commit & Write dev commit changes.
	export FSM_VERSION=$(IMAGE_VERSION)-dev && $(KUSTOMIZE) build config/overlays/dev/ | envsubst > $(DEV_ARTIFCAT_YAML)

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
	curl -sSfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh -s -- -b $(TOOLS_DIR)
	GORELEASER_CURRENT_TAG=$(GIT_VERSION) $(TOOLS_DIR)/goreleaser release --rm-dist --parallelism 5


.PHONY: pre-release
pre-release: check_release_version manifests generate fmt vet kustomize edit_image  ## Create release commit changes to commit & Write release commit changes.
	export FSM_VERSION=$(RELEASE_VERSION) && $(KUSTOMIZE) build config/overlays/release/ | envsubst > $(RELEASE_ARTIFCAT_YAML)
	echo "Replacing image tag to $(subst v,,$(IMAGE_VERSION))"
	sed -i '' 's/proxy-init:latest/fsm-proxy-init:$(subst v,,$(IMAGE_VERSION))/g' $(RELEASE_ARTIFCAT_YAML)
	sed -i '' 's/cluster-connector:latest/fsm-cluster-connector:$(subst v,,$(IMAGE_VERSION))/g' $(RELEASE_ARTIFCAT_YAML)

.PHONY: edit_image
edit_image: $(foreach i,$(IMAGE_TARGET_LIST),editimage/$(i))

editimage/%:
	cd config/overlays/release/ && $(KUSTOMIZE) edit set image $*=$(BUILD_IMAGE_REPO)/fsm-$*:$(subst v,,$(IMAGE_VERSION))


.PHONY: release
VERSION_REGEXP := ^v[0-9]+\.[0-9]+\.[0-9]+(\-(alpha|beta|rc)\.[0-9]+)?$
release: ## Create a release tag, push to git repository and trigger the release workflow.
ifeq (,$(RELEASE_VERSION))
	$(error "RELEASE_VERSION must be set to tag HEAD")
endif
ifeq (,$(shell [[ "$(RELEASE_VERSION)" =~ $(VERSION_REGEXP) ]] && echo 1))
	$(error "Version $(RELEASE_VERSION) must match regexp $(VERSION_REGEXP)")
endif
	git tag --sign --message "flomesh-service-mesh $(RELEASE_VERSION)" $(RELEASE_VERSION)
	git verify-tag --verbose $(RELEASE_VERSION)
	git push origin --tags

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.2)

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
