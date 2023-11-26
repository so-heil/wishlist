# syntax=docker/dockerfile:1
FROM golang:1.21.4 AS build
ENV CGO_ENABLED 0
ENV GOOS linux
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o wishapi cmd/wishapi/main.go

FROM scratch AS deploy
COPY --from=build /app/wishapi /wishapi
ENTRYPOINT ["/wishapi"]