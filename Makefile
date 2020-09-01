.PHONY: all node-mock

IMAGE_NAME=$(if $(ENV_IMAGE_NAME),$(ENV_IMAGE_NAME),hydro-monitor/node-mock)
IMAGE_VERSION=$(if $(ENV_IMAGE_VERSION),$(ENV_IMAGE_VERSION),v0.0.0)

$(info node-mock image settings: $(IMAGE_NAME) version $(IMAGE_VERSION))

all: node-mock

test:
	go test github.com/hydro-monitor/node-mock/pkg/... -cover
	go vet github.com/hydro-monitor/node-mock/pkg/...

node-mock:
	go build -o _output/node-mock ./cmd

image-node-mock:
	go mod vendor
	docker build -t $(IMAGE_NAME):$(IMAGE_VERSION) -f deploy/docker/Dockerfile .

push-image-node-mock: image-node-mock
	docker push $(IMAGE_NAME):$(IMAGE_VERSION)

clean:
	go clean -r -x
	rm -f deploy/docker/node-mock
