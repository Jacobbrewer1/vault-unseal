FROM golang:alpine as build
WORKDIR /build

COPY go.sum go.mod /build/

RUN go mod download
RUN go mod tidy

COPY . /build/
RUN go build -o vault-unseal .

FROM ubuntu:latest

COPY --from=build /build/vault-unseal /usr/local/bin/vault-unseal
ENV PATH="/usr/local/bin:${PATH}"

ENTRYPOINT ["vault-unseal"]
