FROM        golang
MAINTAINER  The Prometheus Authors <prometheus-developers@googlegroups.com>

FROM golang:alpine as build
WORKDIR /go/src/github.com/prometheus/prom2json/lib
COPY lib .
WORKDIR /go/src/github.com/prometheus/prom2json/cmd/prom2json/
COPY cmd/prom2json .
RUN apk --update add git \
 && go get \
 && go build

FROM alpine 
COPY --from=build /go/src/github.com/prometheus/prom2json/cmd/prom2json/prom2json /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/prom2json"]
