BINDIR ?= $(CURDIR)/bin
ARCH   ?= amd64

help:  ## display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: help build docker all clean

build: ## build version-checkers
	mkdir -p $(BINDIR)
	CGO_ENABLED=0 go build -o ./bin/version-checker ./cmd/.

docker: ## build docker image
	GOARCH=$(ARCH) GOOS=linux CGO_ENABLED=0 go build -o ./bin/version-checker-linux ./cmd/.
	docker build -t gcr.io/jetstack-cre/version-checker:v0.0.1-alpha.0 .

clean: ## clean up created files
	rm -rf \
		$(BINDIR)

all: build docker ## runs build and docker
