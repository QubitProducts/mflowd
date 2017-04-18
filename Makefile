.PHONY: bootstrap mflowd test clean

all: mflowd

install:
	glide install

mflowd:
	go build

test:
	go test

clean:
	go clean
