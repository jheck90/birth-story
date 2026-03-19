IMAGE   := jheck90/birth-story
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD)

.PHONY: build push release

build:
	docker build -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

push:
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest

release: build push
