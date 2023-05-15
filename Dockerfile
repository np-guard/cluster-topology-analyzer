# FROM golang:1.19-alpine
FROM golang@sha256:685a22e459f9516f27d975c5cc6accc11223ee81fdfbbae60e39cc3b87357306

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
