IMAGE  := jheck90/birth-story
TAG    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "latest")

.PHONY: build push release

build:
	docker build -t $(IMAGE):$(TAG) -t $(IMAGE):latest .

push:
	docker push $(IMAGE):$(TAG)
	docker push $(IMAGE):latest

release: build push
