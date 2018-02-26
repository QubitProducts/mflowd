FROM golang:1.10-alpine


RUN mkdir -p /go/src/github.com/QubitProducts/mflowd
RUN mkdir -p /etc/mflowd
WORKDIR /go/src/github.com/QubitProducts/mflowd

ADD . /go/src/github.com/QubitProducts/mflowd
ADD misc/run_mflowd.sh run_mflowd.sh
RUN go build
CMD ["/bin/sh", "run_mflowd.sh"]
