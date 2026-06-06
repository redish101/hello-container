FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server .

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

# 拷贝二进制
COPY --from=builder /app/server .

# 如果你的 html template 在文件夹里（很重要）
COPY --from=builder /app/templates ./templates

# 端口（按你项目改）
EXPOSE 8080

# 启动
CMD ["./server"]