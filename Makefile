#!make

TARGETS      := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64
BINNAME      ?= ecnet
DIST_DIRS    := find * -type d -exec
CTR_REGISTRY ?= flomesh
CTR_TAG      ?= latest
VERIFY_TAGS  ?= false

GOPATH = $(shell go env GOPATH)
GOBIN  = $(GOPATH)/bin
GOX    = go run github.com/mitchellh/gox
SHA256 = sha256sum
ifeq ($(shell uname),Darwin)
	SHA256 = shasum -a 256
endif

VERSION ?= dev
BUILD_DATE ?=
GIT_SHA=$$(git rev-parse HEAD)
BUILD_DATE_VAR := github.com/flomesh-io/ErieCanal/pkg/ecnet/version.BuildDate
BUILD_VERSION_VAR := github.com/flomesh-io/ErieCanal/pkg/ecnet/version.Version
BUILD_GITCOMMIT_VAR := github.com/flomesh-io/ErieCanal/pkg/ecnet/version.GitCommit
DOCKER_GO_VERSION = 1.19
DOCKER_BUILDX_PLATFORM ?= linux/amd64
# Value for the --output flag on docker buildx build.
# https://docs.docker.com/engine/reference/commandline/buildx_build/#output
DOCKER_BUILDX_OUTPUT ?= type=registry

LDFLAGS ?= "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(VERSION) -X $(BUILD_GITCOMMIT_VAR)=$(GIT_SHA) -s -w"

# These two values are combined and passed to go test
E2E_FLAGS ?= -installType=KindCluster
E2E_FLAGS_DEFAULT := -test.v -ginkgo.v -ginkgo.progress -ctrRegistry $(CTR_REGISTRY) -ecnetImageTag $(CTR_TAG)

# Installed Go version
# This is the version of Go going to be used to compile this project.
# It will be compared with the minimum requirements for ECNET.
GO_VERSION_MAJOR = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1)
GO_VERSION_MINOR = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)
GO_VERSION_PATCH = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f3)
ifeq ($(GO_VERSION_PATCH),)
GO_VERSION_PATCH := 0
endif

check-env:
ifndef CTR_REGISTRY
	$(error CTR_REGISTRY environment variable is not defined; see the .env.example file for more information; then source .env)
endif
ifndef CTR_TAG
	$(error CTR_TAG environment variable is not defined; see the .env.example file for more information; then source .env)
endif

.PHONY: build-ecnet
build-ecnet: helm-update-dep cmd/ecnet/cli/chart.tgz
	CGO_ENABLED=0 go build -v -o ./bin/ecnet -ldflags ${LDFLAGS} ./cmd/ecnet/cli

cmd/ecnet/cli/chart.tgz: scripts/generate_chart/generate_chart.go $(shell find charts/ecnet)
	go run $< > $@

helm-update-dep:
	helm dependency update charts/ecnet/

.PHONY: clean-ecnet
clean-ecnet:
	@rm -rf bin/ecnet

.PHONY: codegen
codegen:
	./codegen/gen-crd-client.sh

.PHONY: chart-readme
chart-readme:
	go run github.com/norwoodj/helm-docs/cmd/helm-docs -c charts -t charts/ecnet/README.md.gotmpl

.PHONY: chart-check-readme
chart-check-readme: chart-readme
	@git diff --exit-code charts/ecnet/README.md || { echo "----- Please commit the changes made by 'make chart-readme' -----"; exit 1; }

.PHONY: helm-lint
helm-lint:
	@helm lint charts/ecnet/ || { echo "----- Schema validation failed for ECNET chart values -----"; exit 1; }

.PHONY: chart-checks
chart-checks: chart-check-readme helm-lint

.PHONY: check-mocks
check-mocks:
	@go run ./mockspec/generate.go
	@git diff --exit-code || { echo "----- Please commit the changes made by 'go run ./mockspec/generate.go' -----"; exit 1; }

.PHONY: check-codegen
check-codegen:
	@./codegen/gen-crd-client.sh
	@git diff --exit-code || { echo "----- Please commit the changes made by './codegen/gen-crd-client.sh' -----"; exit 1; }

.PHONY: go-checks
go-checks: go-lint go-fmt go-mod-tidy check-mocks check-codegen

.PHONY: go-vet
go-vet:
	go vet ./...

.PHONY: go-lint
go-lint: embed-files-test
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v1.50 golangci-lint run --config .golangci.yml

.PHONY: go-fmt
go-fmt:
	go fmt ./...

.PHONY: go-mod-tidy
go-mod-tidy:
	./scripts/go-mod-tidy.sh

.PHONY: go-test
go-test: helm-update-dep cmd/ecnet/cli/chart.tgz
	./scripts/go-test.sh

.PHONY: go-test-coverage
go-test-coverage: embed-files
	./scripts/test-w-coverage.sh

.PHONY: go-benchmark
go-benchmark: embed-files
	./scripts/go-benchmark.sh

lint-c:
	clang-format --Werror -n bpf/*.c bpf/headers/*.h

format-c:
	find . -regex '.*\.\(c\|h\)' -exec clang-format -style=file -i {} \;

.PHONY: kind-up
kind-up:
	./scripts/kind-with-registry.sh

.PHONY: kind-reset
kind-reset:
	kind delete cluster --name ecnet

.env:
	cp .env.example .env

.PHONY: docker-build-ecnet-controller
docker-build-ecnet-controller:
	docker buildx build --builder ecnet --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/ecnet-controller:$(CTR_TAG) -f dockerfiles/Dockerfile.ecnet-controller --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg LDFLAGS=$(LDFLAGS) .

.PHONY: docker-build-ecnet-crds
docker-build-ecnet-crds:
	docker buildx build --builder ecnet --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/ecnet-crds:$(CTR_TAG) -f dockerfiles/Dockerfile.ecnet-crds .

.PHONY: docker-build-ecnet-bootstrap
docker-build-ecnet-bootstrap:
	docker buildx build --builder ecnet --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/ecnet-bootstrap:$(CTR_TAG) -f dockerfiles/Dockerfile.ecnet-bootstrap --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg LDFLAGS=$(LDFLAGS) .

.PHONY: docker-build-ecnet-preinstall
docker-build-ecnet-preinstall:
	docker buildx build --builder ecnet --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/ecnet-preinstall:$(CTR_TAG) -f dockerfiles/Dockerfile.ecnet-preinstall --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg LDFLAGS=$(LDFLAGS) .

.PHONY: docker-build-ecnet-bridge
docker-build-ecnet-bridge:
	docker buildx build --builder ecnet --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/ecnet-bridge:$(CTR_TAG) -f dockerfiles/Dockerfile.ecnet-bridge --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg LDFLAGS=$(LDFLAGS) .

ECNET_TARGETS = ecnet-crds ecnet-bootstrap ecnet-preinstall ecnet-controller ecnet-bridge
DOCKER_ECNET_TARGETS = $(addprefix docker-build-, $(ECNET_TARGETS))

.PHONY: docker-build-ecnet
docker-build-ecnet: $(DOCKER_ECNET_TARGETS)

.PHONY: buildx-context
buildx-context:
	@if ! docker buildx ls | grep -q "^ecnet "; then docker buildx create --name ecnet --driver-opt network=host; fi

check-image-exists-%: NAME=$(@:check-image-exists-%=%)
check-image-exists-%:
	@if [ "$(VERIFY_TAGS)" = "true" ]; then scripts/image-exists.sh $(CTR_REGISTRY)/$(NAME):$(CTR_TAG); fi

$(foreach target,$(ECNET_TARGETS),$(eval docker-build-$(target): check-image-exists-$(target) buildx-context))

docker-digest-%: NAME=$(@:docker-digest-%=%)
docker-digest-%:
	@docker buildx imagetools inspect $(CTR_REGISTRY)/$(NAME):$(CTR_TAG) --raw | $(SHA256) | awk '{print "$(NAME): sha256:"$$1}'

.PHONY: docker-digests-ecnet
docker-digests-ecnet: $(addprefix docker-digest-, $(ECNET_TARGETS))

.PHONY: docker-build
docker-build: docker-build-ecnet

.PHONY: docker-build-cross-ecnet docker-build-cross
docker-build-cross-ecnet: DOCKER_BUILDX_PLATFORM=linux/amd64,linux/arm64
docker-build-cross-ecnet: docker-build-ecnet
docker-build-cross: docker-build-cross-ecnet

.PHONY: embed-files
embed-files: helm-update-dep cmd/ecnet/cli/chart.tgz

.PHONY: embed-files-test
embed-files-test:
	./scripts/generate-dummy-embed.sh

.PHONY: build-ci
build-ci: embed-files
	go build -v ./...

.PHONY: trivy-ci-setup
trivy-ci-setup:
	wget https://github.com/aquasecurity/trivy/releases/download/v0.23.0/trivy_0.23.0_Linux-64bit.tar.gz
	tar zxvf trivy_0.23.0_Linux-64bit.tar.gz
	echo $$(pwd) >> $(GITHUB_PATH)

# Show all vulnerabilities in logs
trivy-scan-verbose-%: NAME=$(@:trivy-scan-verbose-%=%)
trivy-scan-verbose-%:
	trivy image "$(CTR_REGISTRY)/$(NAME):$(CTR_TAG)"

# Exit if vulnerability exists
trivy-scan-fail-%: NAME=$(@:trivy-scan-fail-%=%)
trivy-scan-fail-%:
	trivy image --exit-code 1 --ignore-unfixed --severity MEDIUM,HIGH,CRITICAL "$(CTR_REGISTRY)/$(NAME):$(CTR_TAG)"

.PHONY: trivy-scan-images trivy-scan-images-fail trivy-scan-images-verbose
trivy-scan-images-verbose: $(addprefix trivy-scan-verbose-, $(ECNET_TARGETS))
trivy-scan-images-fail: $(addprefix trivy-scan-fail-, $(ECNET_TARGETS))
trivy-scan-images: trivy-scan-images-verbose trivy-scan-images-fail

.PHONY: shellcheck
shellcheck:
	shellcheck -x $(shell find . -name '*.sh')

.PHONY: install-git-pre-push-hook
install-git-pre-push-hook:
	./scripts/install-git-pre-push-hook.sh

# -------------------------------------------
#  release targets below
# -------------------------------------------

.PHONY: build-cross
build-cross: helm-update-dep cmd/ecnet/cli/chart.tgz
	GO111MODULE=on CGO_ENABLED=0 $(GOX) -ldflags $(LDFLAGS) -parallel=5 -output="_dist/{{.OS}}-{{.Arch}}/$(BINNAME)" -osarch='$(TARGETS)' ./cmd/ecnet/cli

.PHONY: dist
dist:
	( \
		cd _dist && \
		$(DIST_DIRS) cp ../LICENSE {} \; && \
		$(DIST_DIRS) cp ../README.md {} \; && \
		$(DIST_DIRS) tar -zcf erie-canal-net-${VERSION}-{}.tar.gz {} \; && \
		$(DIST_DIRS) zip -r erie-canal-net-${VERSION}-{}.zip {} \; && \
		$(SHA256) erie-canal-net-* > sha256sums.txt \
	)

.PHONY: release-artifacts
release-artifacts: build-cross dist
