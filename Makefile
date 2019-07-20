DOCKER_REPO ?= quay.io/freshtracks.io/avalanche
DOCKER_TAG ?= $(shell git rev-parse --abbrev-ref HEAD)-$(shell date -u +"%Y-%m-%d")-$(shell git rev-parse --short HEAD)

.PHONY: docker-build
docker-build:
	docker build . -t $(DOCKER_REPO):$(DOCKER_TAG)

.PHONY: docker-publish
docker-publish: docker-build
	docker push $(DOCKER_REPO):$(DOCKER_TAG)
