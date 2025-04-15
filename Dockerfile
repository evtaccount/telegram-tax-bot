FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod . 
COPY go.sum . 
RUN go mod download

COPY . .

# Кросс-компиляция под Linux x86_64
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -o bot main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/bot .
COPY --from=builder /app/data ./data

RUN mkdir -p /root/logs

# Точка входа
CMD ["./bot"]
