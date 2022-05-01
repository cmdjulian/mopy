# syntax = docker/dockerfile:1.4
# compile go app
FROM --platform=$BUILDPLATFORM golang:1.18-alpine AS builder
ENV CGO_ENABLED=0
WORKDIR /build
RUN --mount=type=cache,target=/etc/apk/cache apk --update-cache add upx tzdata
ARG TARGETOS TARGETARCH
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN --mount=type=bind,target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg \
    go build -ldflags="-s -w" -o /frontend/pydockerfile ./main.go
RUN upx /frontend/pydockerfile

# create image with all required files for squashing in later stage
FROM scratch AS squash
COPY --link --from=builder /etc/passwd /etc/group /etc/
COPY --link --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --link --from=builder /usr/share/zoneinfo/Europe/Berlin /usr/share/zoneinfo/Europe/Berlin
COPY --link --from=builder --chown=65534:65534 /frontend/pydockerfile /frontend/pydockerfile

# final image
FROM scratch
LABEL org.opencontainers.image.url="https://gitlab.com/cmdjulian/pydockerfile" \
      org.opencontainers.image.source="https://gitlab.com/cmdjulian/pydockerfile" \
      org.opencontainers.image.version="v1" \
      org.opencontainers.image.title="PyDockerfile" \
      org.opencontainers.image.description="Buildkit frontend for building Python Docker Images" \
      org.opencontainers.image.documentation="https://gitlab.com/cmdjulian/pydockerfile" \
      org.opencontainers.image.authors="cmdjulian" \
      org.opencontainers.image.licenses="MIT"
ENV TZ=Europe/Berlin SSL_CERT_DIR=/etc/ssl/certs PATH=/frontend
USER 65534:65534
COPY --link --from=squash / /

ENTRYPOINT ["/frontend/pydockerfile"]