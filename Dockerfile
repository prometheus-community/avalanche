FROM golang:1.12 AS mod
WORKDIR $GOPATH/avalanche
COPY go.mod .
COPY go.sum .
RUN GO111MODULE=on go mod download

FROM golang:1.12 as build
COPY --from=mod $GOCACHE $GOCACHE
COPY --from=mod $GOPATH/pkg/mod $GOPATH/pkg/mod
WORKDIR $GOPATH/avalanche
COPY . .
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -o=/bin/avalanche ./cmd

FROM scratch
COPY --from=build /bin/avalanche /bin/avalanche
EXPOSE 9001
ENTRYPOINT ["/bin/avalanche"]
