FROM golang:1.9
WORKDIR /go/src/github.com/Fresh-Tracks/avalanche
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o=/bin/avalanche ./cmd

FROM scratch
COPY --from=0 /bin/avalanche /bin/avalanche
EXPOSE 9001
ENTRYPOINT ["/bin/avalanche"]
