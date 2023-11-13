# FROM golang:1.20-alpine
FROM golang@sha256:81cd210ae58a6529d832af2892db822b30d84f817a671b8e1c15cff0b271a3db

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
