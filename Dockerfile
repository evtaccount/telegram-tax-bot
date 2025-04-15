FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod . 
COPY go.sum . 
RUN go mod download
COPY . .
RUN go build -o bot main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/bot .
COPY --from=builder /app/data ./data
RUN mkdir -p /root/logs
ENV TELEGRAM_BOT_TOKEN=""
CMD ["./bot"]
