# 第一阶段：构建阶段
FROM golang:1.21 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .

# 第二阶段：运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .

# ENV APP_ENV=production
EXPOSE 8080

CMD ["./main"]
