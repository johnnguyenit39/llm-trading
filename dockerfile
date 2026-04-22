FROM golang:1.23.4-alpine AS builder

ENV GOARCH=amd64
ENV GOOS=linux

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main .

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/main .

CMD ["./main"]
