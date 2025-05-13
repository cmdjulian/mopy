# syntax = docker/dockerfile:1.5.2

FROM --platform=$BUILDPLATFORM golang:1.24.3-alpine AS builder
WORKDIR /build
RUN --mount=type=cache,target=/etc/apk/cache apk --update-cache add tzdata
COPY --link . .
ARG TARGETOS TARGETARCH
ENV GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg \
    go build -ldflags="-s -w" -o /frontend/mopy ./cmd/mopy/main.go


FROM --platform=$BUILDPLATFORM scratch

ENV TZ=Europe/Berlin SSL_CERT_DIR=/etc/ssl/certs PATH=/frontend
USER 65534:65534

COPY --link --from=builder /etc/passwd /etc/group /etc/
COPY --link --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --link --from=builder --chown=65534:65534 /frontend/mopy /frontend/mopy

ENTRYPOINT ["/frontend/mopy"]
