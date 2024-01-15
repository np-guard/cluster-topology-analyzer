# FROM golang:1.20-alpine
FROM golang@sha256:f9593279431875e29d178bea563344f0fb0e592adc72e8ae24bdc3b444da1e42

RUN apk update && apk upgrade && apk --no-cache add make

WORKDIR /go/src/github.ibm.com/gitsecure-net-top/

COPY pkg/    pkg/
COPY cmd/    cmd/
COPY go.mod go.sum Makefile ./

RUN make

FROM registry.access.redhat.com/ubi8/ubi@sha256:1fdb97f2d2a44fdef3feaa69100f154631bae65130105ac685d0e34eb1d8c3d0
RUN yum -y upgrade

WORKDIR /gitsecure
COPY --from=0 go/src/github.ibm.com/gitsecure-net-top/bin/net-top .

ENTRYPOINT ["/gitsecure/net-top"]
