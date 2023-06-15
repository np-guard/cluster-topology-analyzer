# FROM golang:1.19-alpine
FROM golang@sha256:519184356f970849c5f58764014c13ce20c3a3cadc13be7b8d87269bc5554ccd

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
