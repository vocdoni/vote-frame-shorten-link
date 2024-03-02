FROM golang:1.22 AS builder

WORKDIR /src
ENV CGO_ENABLED=0
RUN go env -w GOCACHE=/go-cache
COPY . .
RUN --mount=type=cache,target=/go-cache go mod download
RUN --mount=type=cache,target=/go-cache go build -o=shorten -ldflags="-s -w"

FROM debian:bookworm-slim as base

WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app
COPY --from=builder /src/shorten ./

ENTRYPOINT ["/app/shorten"]
