# syntax = docker/dockerfile:1.4.3

FROM --platform=$BUILDPLATFORM golang:1.20.0-alpine AS builder
WORKDIR /build
ARG TARGETOS TARGETARCH
ENV GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg --mount=source=.,target=. \
    go build -ldflags="-s -w" -o /frontend/mopy ./cmd/mopy/main.go


FROM gcr.io/distroless/static:nonroot

USER 65532:65532
WORKDIR /home/nonroot
ENTRYPOINT ["/home/nonroot/mopy"]

COPY --link --from=builder --chown=65532:65532 --chmod=500 /frontend/mopy /home/nonroot/mopy
