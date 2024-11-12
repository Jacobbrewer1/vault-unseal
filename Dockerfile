FROM golang:alpine as build
WORKDIR /build

COPY go.sum go.mod Makefile /build/

RUN go mod download
RUN go mod tidy

COPY . /build/
RUN go build -o vault-unseal .

# runtime image
FROM alpine:3.18

RUN apk add --no-cache ca-certificates
COPY --from=build /build/vault-unseal /usr/local/bin/vault-unseal

CMD ["/usr/local/bin/vault-unseal"]
