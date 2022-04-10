FROM golang:alpine AS builder
LABEL stage=gobuilder
ENV CGO_ENABLED 0
RUN apk update --no-cache && apk add --no-cache tzdata upx
WORKDIR /build
COPY . .
RUN --mount=type=cache,target=/go/pkg go build -ldflags="-s -w" -o /app/pydockerfile ./main.go
RUN upx /app/pydockerfile

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo/Europe/Berlin /usr/share/zoneinfo/Europe/Berlin
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
USER 65534:65524
ENV TZ Europe/Berlin
COPY --from=builder --chown=65534:65534 /app /app
WORKDIR /app

ENTRYPOINT ["/app/pydockerfile"]