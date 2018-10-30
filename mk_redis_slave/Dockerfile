FROM golang:1.8 as build

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./
RUN go install -v ./
RUN go build -ldflags "-linkmode external -extldflags -static" -v .

FROM busybox 
COPY --from=build /go/src/app/app /mk_redis_slave
CMD ["/mk_redis_slave"]
