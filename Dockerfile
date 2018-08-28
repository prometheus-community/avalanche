FROM golang:1.9-alpine

WORKDIR /go/src/github.com/Fresh-Tracks/avalanche
COPY . .

ENV CGO_ENABLED=0

RUN go build -o=/bin/avalanche ./cmd

ENTRYPOINT ["/bin/avalanche"]