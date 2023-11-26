# syntax=docker/dockerfile:1
FROM golang:1.21.4 AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=linux go build -o ./binary cmd/wishapi/main.go

FROM alpine:3.18.4 AS deploy
WORKDIR /app
COPY --from=build /app/binary ./
ENTRYPOINT /app/binary