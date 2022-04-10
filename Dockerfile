# compile app
FROM golang:alpine AS builder
LABEL stage=gobuilder
ENV CGO_ENABLED=0
RUN apk update --no-cache && apk add --no-cache tzdata upx
WORKDIR /build
COPY . .
RUN --mount=type=cache,target=/go/pkg --mount=type=cache,target=/root/.cache/go-build go build -ldflags="-s -w" -o /app/pydockerfile ./main.go
RUN upx /app/pydockerfile

# Generate latest ca-certificates
FROM debian:buster-slim AS certs
RUN apt update && apt install -y ca-certificates && cat /etc/ssl/certs/* > /ca-certificates.crt

# final image
FROM scratch
COPY --from=builder /usr/share/zoneinfo/Europe/Berlin /usr/share/zoneinfo/Europe/Berlin
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=certs /ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER 65534:65524
ENV TZ=Europe/Berlin USER=nobody SSL_CERT_DIR=/etc/ssl/certs PATH=/app
COPY --from=builder --chown=65534:65534 /app /app
WORKDIR /app

ENTRYPOINT ["/app/pydockerfile"]