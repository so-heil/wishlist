FROM golang:1.21.4 AS base
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

FROM base AS live
RUN go install github.com/cosmtrek/air@latest
ENTRYPOINT ["air", "-c", ".air.toml"]

FROM base AS build
ENV CGO_ENABLED 0
ENV GOOS linux

RUN go build -o admin cmd/admin/main.go
RUN go build -o wishapi cmd/wishapi/main.go

FROM alpine:3.18.4 AS admin
COPY --from=build /app/admin /admin
ENTRYPOINT ["/bin/sh" ]

FROM alpine:3.18.4 AS wishapi
COPY --from=build /app/wishapi /wishapi
ENTRYPOINT ["/wishapi"]