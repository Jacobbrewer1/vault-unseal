FROM golang:alpine as build
WORKDIR /build

COPY go.sum go.mod /build/

RUN go mod download
RUN go mod tidy

COPY . /build/
RUN go build -o vault-unseal .

# runtime image
FROM alpine:3.18

RUN apk add --no-cache ca-certificates
COPY --from=build /build/vault-unseal /usr/local/bin/vault-unseal
ENV PATH="/usr/local/bin:${PATH}"

RUN mkdir -p /tmp/vault/config

# Give the container permissions to read the /tmp/vault directory and all subdirectories
RUN chown -R 1000:1000 /tmp/vault

CMD ["/usr/local/bin/vault-unseal"]
