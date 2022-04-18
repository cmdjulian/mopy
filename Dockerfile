# syntax = docker/dockerfile:1.4
# compile go app
FROM --platform=$BUILDPLATFORM golang:1.16-alpine AS builder
ENV CGO_ENABLED=0
WORKDIR /build
RUN --mount=type=cache,target=/etc/apk/cache apk update && apk add upx tzdata
ARG TARGETOS TARGETARCH
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg <<EOF
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /app/pydockerfile ./main.go
    upx /app/pydockerfile
EOF

# final image
FROM scratch
COPY --link --from=builder /usr/share/zoneinfo/Europe/Berlin /usr/share/zoneinfo/Europe/Berlin
COPY --link --from=builder /etc/passwd /etc/group /etc/
COPY --link --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER 65534:65524
ENV TZ=Europe/Berlin USER=nobody SSL_CERT_DIR=/etc/ssl/certs PATH=/app
WORKDIR /app
COPY --link --from=builder --chown=nobody:nobody /app/pydockerfile .

ENTRYPOINT ["/app/pydockerfile"]