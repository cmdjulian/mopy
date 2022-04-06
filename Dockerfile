# syntax=docker/dockerfile:1.4
FROM golang:1.16-buster AS builder
WORKDIR $GOPATH/src/buildkit-frontend-for-python
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o /go/src/pydockerfile

FROM scratch
COPY --from=builder /go/src/pydockerfile /pydockerfile
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=alpine:latest /etc/passwd /etc/passwd
USER 65534:65524
ENTRYPOINT ["/pydockerfile"]