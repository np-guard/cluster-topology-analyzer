# FROM golang:1.22-alpine
FROM golang@sha256:a7f2fc9834049c1f5df787690026a53738e55fc097cd8a4a93faa3e06c67ee32

RUN apk update && apk upgrade && apk --no-cache add make

WORKDIR /go/src/github.ibm.com/gitsecure-net-top/

COPY pkg/    pkg/
COPY cmd/    cmd/
COPY go.mod go.sum Makefile ./

RUN make

FROM registry.access.redhat.com/ubi9/ubi-minimal@sha256:1b6d711648229a1c987f39cfdfccaebe2bd92d0b5d8caa5dbaa5234a9278a0b2
RUN microdnf --nodocs -y upgrade

WORKDIR /gitsecure
COPY --from=0 go/src/github.ibm.com/gitsecure-net-top/bin/net-top .

ENTRYPOINT ["/gitsecure/net-top"]
