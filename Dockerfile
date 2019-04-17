FROM golang:1.12.1
WORKDIR /go/src/github.com/open-fresh/avalanche
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o=/bin/avalanche ./cmd

FROM scratch
COPY --from=0 /bin/avalanche /bin/avalanche
EXPOSE 9001
ENTRYPOINT ["/bin/avalanche"]
