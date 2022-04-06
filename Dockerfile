# syntax=docker/dockerfile:1

FROM golang:1.16-buster AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o /pydockerfile

##
## Deploy
##
FROM gcr.io/distroless/base-debian10
COPY --from=build /pydockerfile /pydockerfile
USER nonroot:nonroot

ENTRYPOINT ["/pydockerfile"]
