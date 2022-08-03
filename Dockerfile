# FROM golang:1.17-alpine
FROM golang@sha256:99ddec1bbfd6d6bca3f9804c02363daee8c8524dae50df7942e8e60788fd17c9

RUN apk --no-cache add git

WORKDIR /go/src/github.ibm.com/gitsecure-net-top/

COPY pkg/    pkg/
COPY cmd/    cmd/

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build --tags static_all -v -o ./bin/net-top cmd/nettop/main.go

FROM registry.access.redhat.com/ubi8@sha256:6edca3916b34d10481e4d24d14ebe6ebc6db517bec1b2db6ae2d7d47c2ecfaee
RUN yum -y upgrade

WORKDIR /gitsecure
COPY --from=0 go/src/github.ibm.com/gitsecure-net-top/bin/net-top .

ENTRYPOINT ["/gitsecure/net-top"]
