REGISTRY_URI = push.registry.devshift.net
REGISTRY_NS = fabric8-services
REGISTRY_IMAGE = fabric8-jenkins-idler
REGISTRY_URL = ${REGISTRY_URI}/${REGISTRY_NS}/${REGISTRY_IMAGE}
IMAGE_TAG ?= $(shell git rev-parse --short HEAD)

BUILD_DIR = out
PACKAGES = $(shell go list ./...)
SOURCE_DIRS = $(shell echo $(PACKAGES) | awk 'BEGIN{FS="/"; RS=" "}{print $$4}' | uniq)

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

all: tools build test fmtcheck vet image ## Compiles binary and runs format and style checks

build: vendor ## Builds the binary into $GOPATH/bin
	go install ./cmd/fabric8-jenkins-idler

$(BUILD_DIR):
	mkdir $(BUILD_DIR)

$(BUILD_DIR)/$(REGISTRY_IMAGE): vendor $(BUILD_DIR) ## Builds the Linux binary for the container image into $BUILD_DIR
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o $(BUILD_DIR)/$(REGISTRY_IMAGE) ./cmd/fabric8-jenkins-idler

image: $(BUILD_DIR)/$(REGISTRY_IMAGE) ## Builds the container image
	docker build -t $(REGISTRY_URL) -f Dockerfile.deploy .

push: image ## Pushes the container image to the registry
	$(call check_defined, REGISTRY_USER, "You need to pass the registry user via REGISTRY_USER.")
	$(call check_defined, REGISTRY_PASSWORD, "You need to pass the registry password via REGISTRY_PASSWORD.")
	docker login -u $(REGISTRY_USER) -p $(REGISTRY_PASSWORD) $(REGISTRY_URI)
	docker push $(REGISTRY_URL):latest
	docker tag $(REGISTRY_URL):latest $(REGISTRY_URL):$(IMAGE_TAG)
	docker push $(REGISTRY_URL):$(IMAGE_TAG)

tools: tools.timestamp

tools.timestamp:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/golang/lint/golint
	@touch tools.timestamp

vendor: tools.timestamp ## Runs dep to vendor project dependencies
	dep ensure -v

.PHONY: test
test: vendor ## Runs unit tests
	go test -v $(PACKAGES)

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
	@out="$$(golint $(PACKAGES))"; \
	if [ -n "$$out" ]; then \
		echo "$$out"; \
		exit 1; \
	fi

.PHONY: clean
clean: ## Deletes all build artifacts
	rm -rf vendor
	rm -rf tools.timestamp
	rm -rf $(BUILD_DIR)

.PHONY: help
help: ## Prints this help
	@grep -E '^[^.]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-40s\033[0m %s\n", $$1, $$2}'
