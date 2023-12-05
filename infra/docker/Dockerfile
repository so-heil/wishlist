FROM golang:1.21.4 AS build
ENV CGO_ENABLED 0
ENV GOOS linux
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o admin cmd/admin/main.go
RUN go build -o wishapi cmd/wishapi/main.go

FROM alpine:3.18.4 AS admin
COPY --from=build /app/admin /admin
ENTRYPOINT ["/bin/sh" ]

FROM scratch AS wishapi
COPY --from=build /app/wishapi /wishapi
ENTRYPOINT ["/wishapi"]