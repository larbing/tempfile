# 使用 golang 镜像作为构建和运行环境
FROM golang:1.23.4 AS builder

# 设置工作目录
WORKDIR /app

# 拷贝 Go mod 文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 拷贝源代码
COPY . .

# 编译 Go 程序为 Linux 可执行文件
RUN GOOS=linux GOARCH=amd64 go build -o main .

# 启动时执行的命令
CMD ["./main"]
