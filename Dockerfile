# FROM golang:1.20-alpine
FROM golang@sha256:cffaba795c36f07e372c7191b35ceaae114d74c31c3763d442982e3a4df3b39e

RUN apk update && apk upgrade && apk --no-cache add make

WORKDIR /go/src/github.ibm.com/gitsecure-net-top/

COPY pkg/    pkg/
COPY cmd/    cmd/
COPY go.mod go.sum Makefile ./

RUN make

FROM registry.access.redhat.com/ubi8@sha256:b6616b280ec23c2283ac10e19dd3cd4c8e6df14599f6d93f662ca261273097a9
RUN yum -y upgrade

WORKDIR /gitsecure
COPY --from=0 go/src/github.ibm.com/gitsecure-net-top/bin/net-top .

ENTRYPOINT ["/gitsecure/net-top"]
