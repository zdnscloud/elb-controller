FROM golang:1.13.7-alpine3.11 AS build
ENV GOPROXY=https://goproxy.cn
ARG version
ARG buildtime

RUN mkdir -p /go/src/github.com/zdnscloud/elb-controller
COPY . /go/src/github.com/zdnscloud/elb-controller

WORKDIR /go/src/github.com/zdnscloud/elb-controller
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-w -s -X main.version=$version -X main.build=$buildtime" -o cmd/elbc cmd/elbc.go


FROM scratch

LABEL maintainers="Zdns Authors"
LABEL description="Kubernetes external loadbalancer controller"

COPY --from=build /go/src/github.com/zdnscloud/elb-controller/cmd/elbc /elbc

ENTRYPOINT ["/elbc"]
