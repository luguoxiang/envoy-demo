# build stage
FROM golang:alpine AS build-env
RUN apk update
RUN apk add git
RUN apk add curl
ENV PROJECT_DIR /go/src/github.com/luguoxiang/envoy-demo
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh |sh
RUN mkdir -p ${PROJECT_DIR}/cmd
ENV GOPATH /go
WORKDIR ${PROJECT_DIR}
ADD Gopkg.lock .
ADD Gopkg.toml .
RUN dep ensure -vendor-only -v
ADD cmd cmd
ADD pkg pkg
RUN go build -o envoy_server cmd/envoy_server.go

# final stage
FROM golang:alpine
WORKDIR /app
COPY --from=build-env /go/src/github.com/luguoxiang/envoy-demo/envoy_server /app/
ENV https_proxy ""
ENV http_proxy ""
CMD ./envoy_server -alsologtostderr

