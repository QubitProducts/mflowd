.PHONY: mflowd test clean

all: mflowd

bootstrap:
	dep ensure -v

mflowd:
	go build

test:
	go test

vendor:
	make bootstrap

docker: vendor
	docker build -t mflowd .

clean:
	go clean
