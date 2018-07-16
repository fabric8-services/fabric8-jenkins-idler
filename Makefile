REGISTRY_URI = quay.io
REGISTRY_NS = fabric8-services
REGISTRY_IMAGE = fabric8-jenkins-idler

ifeq ($(TARGET),rhel)
	DOCKERFILE_DEPLOY := Dockerfile.deploy.rhel
	REGISTRY_URL = ${REGISTRY_URI}/openshiftio/rhel-${REGISTRY_NS}-${REGISTRY_IMAGE}
else
	DOCKERFILE_DEPLOY := Dockerfile.deploy
	REGISTRY_URL = ${REGISTRY_URI}/openshiftio/${REGISTRY_NS}-${REGISTRY_IMAGE}
endif

IMAGE_TAG ?= $(shell git rev-parse --short HEAD)

BUILD_DIR = out
PACKAGES = $(shell go list ./...)
LINT_PACKAGES = $(shell echo $(PACKAGES) | sed -e 's@github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration@@')
SOURCE_DIRS = $(shell echo $(PACKAGES) | awk 'BEGIN{FS="/"; RS=" "}{print $$4}' | uniq)
LD_FLAGS := -X github.com/fabric8-services/fabric8-jenkins-idler/internal/version.version=$(IMAGE_TAG)

# Goa
AUTH_GEN_DIR=internal/auth/client

# Misc
.DEFAULT_GOAL := help

# Check that given variables are set and all have non-empty values,
# die with an error otherwise.
#
# Params:
#   1. Variable name(s) to test.
#   2. (optional) Error message to print.
check_defined = \
    $(strip $(foreach 1,$1, \
        $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = \
    $(if $(value $1),, \
      $(error Undefined $1$(if $2, ($2))))

.PHONY: all
all: tools build test fmtcheck vet lint validate_commits image ## Compiles binary and runs format and style checks

build: vendor $(AUTH_GEN_DIR)/*.go ## Builds the binary into $GOPATH/bin
	go install -ldflags="$(LD_FLAGS)" ./cmd/fabric8-jenkins-idler

$(BUILD_DIR):
	@mkdir $(BUILD_DIR)

$(BUILD_DIR)/$(REGISTRY_IMAGE): vendor  $(AUTH_GEN_DIR)/*.go $(BUILD_DIR) ## Builds the Linux binary for the container image into $BUILD_DIR
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build  -ldflags="$(LD_FLAGS)" -o $(BUILD_DIR)/$(REGISTRY_IMAGE) ./cmd/fabric8-jenkins-idler

login:
	$(call check_defined, REGISTRY_USER, "You need to pass the registry user via REGISTRY_USER.")
	$(call check_defined, REGISTRY_PASSWORD, "You need to pass the registry password via REGISTRY_PASSWORD.")
	docker login -u $(REGISTRY_USER) -p $(REGISTRY_PASSWORD) $(REGISTRY_URI)

image: $(BUILD_DIR)/$(REGISTRY_IMAGE) ## Builds the container image
	docker build -t $(REGISTRY_URL) -f $(DOCKERFILE_DEPLOY) .

push: image ## Pushes the container image to the registry
	docker push $(REGISTRY_URL):latest
	docker tag $(REGISTRY_URL):latest $(REGISTRY_URL):$(IMAGE_TAG)
	docker push $(REGISTRY_URL):$(IMAGE_TAG)

tools: tools.timestamp

tools.timestamp:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/golang/lint/golint || true
	go get -u github.com/vbatts/git-validation/...
	go get -u github.com/goadesign/goa/goagen
	go get -u github.com/haya14busa/goverage
	@touch tools.timestamp

vendor: tools.timestamp $(AUTH_GEN_DIR)/*.go ## Runs dep to vendor project dependencies
	dep ensure -v

$(AUTH_GEN_DIR)/*.go:  ## Runs goagen to generate auth service client
	goagen client -d github.com/fabric8-services/fabric8-auth/design --notool --out internal/auth --pkg client

.PHONY: test
test: vendor ## Runs unit tests
	@go test $(PACKAGES)

.PHONY: coverage
coverage: vendor tools $(BUILD_DIR) ## Run coverage, need goverage tool installed
	goverage -coverprofile=$(BUILD_DIR)/coverage.out $(PACKAGES) && \
	go tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo $(realpath $(BUILD_DIR))/coverage.html

.PHONY: fmtcheck
fmtcheck: ## Runs gofmt and returns error in case of violations
	@gofmt -l -s $(SOURCE_DIRS) | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi

.PHONY: fmt
fmt: ## Runs gofmt and formats code violations
	@gofmt -l -s -w $(SOURCE_DIRS)

.PHONY: vet
vet: ## Runs 'go vet' for common coding mistakes
	@go vet $(PACKAGES)

.PHONY: lint
lint: ## Runs golint
	@out="$$(golint $(LINT_PACKAGES))"; \
	if [ -n "$$out" ]; then \
		echo "$$out"; \
		exit 1; \
	fi

.PHONY: validate_commits
validate_commits: tools ## Validates git commit messages
ifeq ($(origin TRAVIS_COMMIT_RANGE), undefined)
	git-validation  \
			-run short-subject,message_regexp='^(Revert\s*)?(Fix\s*)?(Issue\s*)?#[0-9]+ [A-Z]+.*' \
			-range `git rev-parse master@{u}`...HEAD
else
	git-validation  \
			-run short-subject,message_regexp='^(Revert\s*)?(Fix\s*)?(Issue\s*)?#[0-9]+ [A-Z]+.*' \
			-range $$TRAVIS_COMMIT_RANGE

endif

.PHONY: clean
clean: ## Deletes all build artifacts
	rm -rf vendor
	rm -rf tools.timestamp
	rm -rf $(BUILD_DIR)

.PHONY: help
help: ## Prints this help
	@grep -E '^[^.]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-40s\033[0m %s\n", $$1, $$2}'
