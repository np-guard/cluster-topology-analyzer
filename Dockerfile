# FROM golang:1.20-alpine
FROM golang@sha256:03278bc16e1a5b4fb6cdd3462108c060aa1e9c2353ce4d15d744b3c40168677d

RUN apk update && apk upgrade && apk --no-cache add make

WORKDIR /go/src/github.ibm.com/gitsecure-net-top/

COPY pkg/    pkg/
COPY cmd/    cmd/
COPY go.mod go.sum Makefile ./

RUN make

FROM registry.access.redhat.com/ubi8@sha256:269e9753043a4066af12649e921c6ad3201702fda5b2652f7a4aa010c2ed4c1a
RUN yum -y upgrade

WORKDIR /gitsecure
COPY --from=0 go/src/github.ibm.com/gitsecure-net-top/bin/net-top .

ENTRYPOINT ["/gitsecure/net-top"]
