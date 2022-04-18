# syntax = docker/dockerfile:1.4
# compile go app
FROM --platform=$BUILDPLATFORM golang:1.16-alpine AS builder
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/etc/apk/cache apk update && apk add upx tzdata
WORKDIR /build
COPY --link . .
RUN --mount=type=cache,target=/go/pkg go get -u ./... && go mod download
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build <<EOF
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /app/pydockerfile ./main.go
    upx /app/pydockerfile
EOF

# final image
FROM scratch
COPY --from=builder /usr/share/zoneinfo/Europe/Berlin /usr/share/zoneinfo/Europe/Berlin
COPY --from=builder /etc/passwd /etc/group /etc/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER 65534:65524
ENV TZ=Europe/Berlin USER=nobody SSL_CERT_DIR=/etc/ssl/certs PATH=/app
WORKDIR /app
COPY --from=builder --chown=65534:65534 /app /app

ENTRYPOINT ["/app/pydockerfile"]