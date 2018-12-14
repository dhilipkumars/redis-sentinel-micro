FROM golang:1.11.3-alpine3.8 as build

RUN apk add --no-cache git

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./
RUN go install -v ./

RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -v .

FROM busybox 
COPY --from=build /go/src/app/app /redis_sentinel_k8s
ENTRYPOINT ["/redis_sentinel_k8s"]
