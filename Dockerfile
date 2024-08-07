# First stage: build the Go binary
FROM golang:1.19 AS build

RUN go env -w GO111MODULE=auto

RUN mkdir /src

WORKDIR /src

COPY go.mod go.sum ./
COPY . /src

WORKDIR /src/cmd/bantool

RUN go get
RUN CGO_ENABLED=0 GOOS=linux go build -o /bantool -v -x

# Second stage: Run Go binary
FROM alpine:3.19 AS build-release-stage

RUN apk update && apk add --no-cache sudo

RUN mkdir /app

WORKDIR /app

COPY --from=build /bantool ./bantool

ENTRYPOINT ["/app/bantool"]
CMD []
