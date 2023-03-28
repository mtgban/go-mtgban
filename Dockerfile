# First stage: build the Go binary
FROM golang:1.19 AS stage-1

RUN go env -w GO111MODULE=auto

WORKDIR /app

COPY . /app

WORKDIR /app/go-mtgban/cmd/bantool

RUN go build -o ../../../bantool.exe

# Second stage: Run the Go binary
FROM golang AS stage-2

RUN go env -w GO111MODULE=auto

RUN apt-get update

RUN apt-get install sudo

RUN sudo apt-get install -y curl xz-utils

RUN mkdir /app

WORKDIR /app

RUN curl "https://mtgjson.com/api/v5/AllPrintings.json.xz" |  xz -dc > allprintings5.json

COPY --from=stage-1 /app/bantool.exe ./bantool.exe

COPY gcp.json gcp.json

COPY .env .env

ENTRYPOINT ["./bantool.exe"]

CMD [""]