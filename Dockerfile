# FROM golang:1.20-alpine
FROM golang@sha256:e9ebfe932adeff65af5338236f0b0604c86b143c1bff3e1d0551d8f6196ab5c5

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
