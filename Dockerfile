# syntax = docker/dockerfile:1.4.3

FROM --platform=$BUILDPLATFORM golang:1.18-alpine AS builder
WORKDIR /build
RUN --mount=type=cache,target=/etc/apk/cache apk --update-cache add tzdata
ARG TARGETOS TARGETARCH
ENV GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0
RUN --mount=type=bind,target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg \
    go build -ldflags="-s -w" -o /frontend/mopy ./cmd/mopy/main.go


FROM --platform=linux/amd64 builder AS shrinker
COPY --from=starudream/upx:latest /usr/bin/upx /usr/bin/upx
RUN upx --best --ultra-brute /frontend/mopy


FROM --platform=$BUILDPLATFORM scratch AS squash
COPY --link --from=builder /etc/passwd /etc/group /etc/
COPY --link --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --link --from=shrinker --chown=65534:65534 /frontend/mopy /frontend/mopy


FROM --platform=$BUILDPLATFORM scratch
LABEL org.opencontainers.image.authors="cmdjulian" \
      org.opencontainers.image.base.name="scratch" \
      org.opencontainers.image.description="Buildkit frontend for building Python Docker Images" \
      org.opencontainers.image.documentation="https://gitlab.com/cmdjulian/mopy/-/blob/main/README.md" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.ref.name="main" \
      org.opencontainers.image.source="https://gitlab.com/cmdjulian/mopy" \
      org.opencontainers.image.title="mopy" \
      org.opencontainers.image.url="https://gitlab.com/cmdjulian/mopy" \
      org.opencontainers.image.vendor="cmdjulian" \
      org.opencontainers.image.version="v1"

ENV TZ=Europe/Berlin SSL_CERT_DIR=/etc/ssl/certs PATH=/frontend
USER 65534:65534
COPY --link --from=squash / /

ENTRYPOINT ["/frontend/mopy"]