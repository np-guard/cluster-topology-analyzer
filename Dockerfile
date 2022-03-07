FROM golang:1.17-alpine

RUN apk --no-cache add git

WORKDIR /go/src/github.ibm.com/gitsecure-net-top/

COPY pkg/    pkg/
COPY cmd/    cmd/

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build --tags static_all -v -o ./bin/net-top cmd/nettop/main.go

FROM registry.access.redhat.com/ubi8
RUN yum -y upgrade

WORKDIR /gitsecure
COPY --from=0 go/src/github.ibm.com/gitsecure-net-top/bin/net-top .

ENTRYPOINT ["/gitsecure/net-top"]
