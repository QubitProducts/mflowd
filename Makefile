.PHONY: mflowd test clean

all: mflowd

bootstrap:
	glide install

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
