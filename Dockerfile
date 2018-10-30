FROM golang:1.8 as build

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./
RUN go install -v ./

RUN go build -ldflags "-linkmode external -extldflags -static" -v .

FROM busybox 
COPY --from=build /go/src/app/app /redis_sentinel_k8s
ENTRYPOINT ["/redis_sentinel_k8s"]
