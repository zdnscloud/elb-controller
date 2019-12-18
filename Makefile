VERSION= v0.0.1
BUILD=`date +%FT%T%z`

LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.build=${BUILD}"
GOSRC = $(shell find . -type f -name '*.go')

build: elb-controller

elb-controller: $(GOSRC) 
	go build ${LDFLAGS} cmd/elbc.go

docker: build-image
	docker push zdnscloud/elb-controller:${VERSION}

build-image:
	docker build -t zdnscloud/elb-controller:${VERSION} --build-arg version=${VERSION} --build-arg buildtime=${BUILD} --no-cache .
	docker image prune -f

clean:
	rm -rf elbc

clean-image:
	docker rmi zdnscloud/elb-controller:${VERSION}
