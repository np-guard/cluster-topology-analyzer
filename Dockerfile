# FROM golang:1.19-alpine
FROM golang@sha256:9668643a2e62d8bd298ef3663a96de4a70ceb2865b9b7cadd1d5e08387745103

RUN apk update && apk upgrade && apk --no-cache add make

WORKDIR /go/src/github.ibm.com/gitsecure-net-top/

COPY pkg/    pkg/
COPY cmd/    cmd/
COPY go.mod go.sum Makefile ./

RUN make

FROM registry.access.redhat.com/ubi8@sha256:4a6dbfbb845810dce5902ab80cb93ecb24c367460fff9d15438e0b3080e244b3
RUN yum -y upgrade

WORKDIR /gitsecure
COPY --from=0 go/src/github.ibm.com/gitsecure-net-top/bin/net-top .

ENTRYPOINT ["/gitsecure/net-top"]
