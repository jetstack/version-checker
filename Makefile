BINDIR ?= $(CURDIR)/bin
ARCH   ?= amd64

help:  ## display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: help build image all clean

deps: ## Download all Dependencies
	go mod download

test: deps ## test version-checker
	go test ./... -coverprofile=coverage.out

$(BINDIR):
	mkdir -p $(BINDIR)

build: deps $(BINDIR) ## build version-checker
	CGO_ENABLED=0 go build -o ./bin/version-checker ./cmd/.

verify: test build ## tests and builds version-checker

image: ## build docker image
	GOARCH=$(ARCH) GOOS=linux CGO_ENABLED=0 go build -o ./bin/version-checker-linux ./cmd/.
	docker build -t quay.io/jetstack/version-checker:v0.5.4 .

clean: ## clean up created files
	rm -rf \
		$(BINDIR)

all: test build image ## runs test, build and image
