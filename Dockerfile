# FROM golang:1.20-alpine
FROM golang@sha256:ae34fbf671566a533f92e5469f3f3d34e9e6fb14c826db09956454da9a84c9a9

RUN apk update && apk upgrade && apk --no-cache add make

WORKDIR /go/src/github.ibm.com/gitsecure-net-top/

COPY pkg/    pkg/
COPY cmd/    cmd/
COPY go.mod go.sum Makefile ./

RUN make

FROM registry.access.redhat.com/ubi8/ubi@sha256:449da7f8f2ef6285a8445a1e31af57a97b9dae5dcf009b1629c59742c89c68c3
RUN yum -y upgrade

WORKDIR /gitsecure
COPY --from=0 go/src/github.ibm.com/gitsecure-net-top/bin/net-top .

ENTRYPOINT ["/gitsecure/net-top"]
