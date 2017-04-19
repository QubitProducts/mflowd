FROM golang:1.7.5-alpine

RUN mkdir -p /go/src/github.com/QubitProducts/mflowd
WORKDIR /go/src/github.com/QubitProducts/mflowd

ADD . /go/src/github.com/QubitProducts/mflowd
ADD misc/run_mflowd.sh run_mflowd.sh
RUN go build
CMD ["/bin/sh", "run_mflowd.sh"]
