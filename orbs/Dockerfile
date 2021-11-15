FROM golang:1.16

COPY . /multi_server

WORKDIR /multi_server

RUN go mod vendor && \
    go build --mod=vendor -gcflags "all=-N -l" -o multi_server .

ENTRYPOINT ["./multi_server"]