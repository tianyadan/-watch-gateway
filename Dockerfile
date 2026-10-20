# ==============================
# 第一阶段：编译 Go
# ==============================
FROM golang:1.26-alpine AS builder

WORKDIR /app

# 先复制依赖文件，利用 Docker 缓存
COPY go.mod go.sum ./

RUN go mod download

# 再复制源码
COPY . .

# 编译 Linux 静态二进制
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /watch-gateway \
    .


# ==============================
# 第二阶段：运行
# ==============================
FROM alpine:3.22

# ca-certificates：
# 你的程序要请求 https://web.chelaile.net.cn
# 以及 https://api.day.app
#
# tzdata：
# 你的代码使用 Asia/Shanghai 时区
RUN apk add --no-cache \
    ca-certificates \
    tzdata

WORKDIR /app

COPY --from=builder /watch-gateway /app/watch-gateway
# 镜像内只带脱敏示例；生产请挂载自己的 config.yaml
COPY config.example.yaml /app/config.example.yaml

EXPOSE 28085

ENTRYPOINT ["/app/watch-gateway", "-config", "/app/config.yaml"]