# FROM golang:1.21-alpine
FROM golang@sha256:d7c6083c5400694f7a17b07c4fad8db9115c0e8e3cf62f781cb29cc890a64e8e

RUN apk update && apk upgrade && apk --no-cache add make

WORKDIR /go/src/github.ibm.com/gitsecure-net-top/

COPY pkg/    pkg/
COPY cmd/    cmd/
COPY go.mod go.sum Makefile ./

RUN make

FROM registry.access.redhat.com/ubi8/ubi@sha256:83068ea81dd02717b8e39b55cdeb2c1b2c9a3db260f01381b991755d44b15073
RUN yum -y upgrade

WORKDIR /gitsecure
COPY --from=0 go/src/github.ibm.com/gitsecure-net-top/bin/net-top .

ENTRYPOINT ["/gitsecure/net-top"]
