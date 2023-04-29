# First stage: build the Go binary
FROM golang:1.19 AS build-stage

RUN go env -w GO111MODULE=auto

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /bantool

# Second stage: Run the Go binary
FROM alpine:latest AS build-release-stage

RUN apk update && apk add --no-cache sudo curl xz

RUN mkdir /app

WORKDIR /app

RUN curl "https://mtgjson.com/api/v5/AllPrintings.json.xz" |  xz -dc > allprintings5.json

COPY --from=build-stage /bantool ./bantool

ENTRYPOINT ["./bantool"]

CMD [""]