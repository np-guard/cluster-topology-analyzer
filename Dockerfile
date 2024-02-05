# FROM golang:1.20-alpine
FROM golang@sha256:f9593279431875e29d178bea563344f0fb0e592adc72e8ae24bdc3b444da1e42

RUN apk update && apk upgrade && apk --no-cache add make

WORKDIR /go/src/github.ibm.com/gitsecure-net-top/

COPY pkg/    pkg/
COPY cmd/    cmd/
COPY go.mod go.sum Makefile ./

RUN make

FROM registry.access.redhat.com/ubi8/ubi@sha256:627867e53ad6846afba2dfbf5cef1d54c868a9025633ef0afd546278d4654eac
RUN yum -y upgrade

WORKDIR /gitsecure
COPY --from=0 go/src/github.ibm.com/gitsecure-net-top/bin/net-top .

ENTRYPOINT ["/gitsecure/net-top"]
