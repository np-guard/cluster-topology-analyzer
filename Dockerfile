# FROM golang:1.19-alpine
FROM golang@sha256:344193a70dc3588452ea39b4a1e465a8d3c91f788ae053f7ee168cebf18e0a50

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
