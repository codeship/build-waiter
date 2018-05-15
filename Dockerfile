# build stage
FROM golang:alpine AS build-env

RUN mkdir -p /go/src/github.com/codeship/build-waiter
COPY . /go/src/github.com/codeship/build-waiter
WORKDIR /go/src/github.com/codeship/build-waiter

RUN go build -o build-waiter

# final stage
FROM alpine
WORKDIR /app
COPY --from=build-env /go/src/github.com/codeship/build-waiter /app/
ENTRYPOINT ./build-waiter