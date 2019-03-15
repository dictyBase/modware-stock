FROM golang:1.11.5-alpine3.8
LABEL maintainer="Siddhartha Basu <siddhartha-basu@northwestern.edu>"
RUN apk add --no-cache git build-base
RUN mkdir -p /modware-stock
WORKDIR /modware-stock
COPY go.mod cmd/modware-stock/main.go ./
ADD internal internal
RUN go get ./... && go mod tidy && go build -o app

FROM alpine:3.7
RUN apk --no-cache add ca-certificates
COPY --from=0 /modware-stock/app /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/app"]