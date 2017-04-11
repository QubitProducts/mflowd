.PHONY: bootstrap mflowd test clean

bootstrap:
	glide install

mflowd:
	go build

test:
	go test

clean:
	go clean
