FROM golang:1.15 as build
WORKDIR $GOPATH/avalanche
COPY . .
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -o=/bin/avalanche ./cmd

FROM scratch
COPY --from=build /bin/avalanche /bin/avalanche
EXPOSE 9001
ENTRYPOINT ["/bin/avalanche"]
