# FROM golang:1.20-alpine
FROM golang@sha256:24a09375a6216764a3eda6a25490a88ac178b5fcb9511d59d0da5ebf9e496474

RUN apk update && apk upgrade && apk --no-cache add make

WORKDIR /go/src/github.ibm.com/gitsecure-net-top/

COPY pkg/    pkg/
COPY cmd/    cmd/
COPY go.mod go.sum Makefile ./

RUN make

FROM registry.access.redhat.com/ubi8@sha256:dd3ea588640fd91ea85f6324df14627b2f8ad52ac55ecd907ca3201cd78667e6
RUN yum -y upgrade

WORKDIR /gitsecure
COPY --from=0 go/src/github.ibm.com/gitsecure-net-top/bin/net-top .

ENTRYPOINT ["/gitsecure/net-top"]
