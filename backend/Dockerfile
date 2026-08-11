# 多阶段构建：编译 → 精简运行
FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o hrms-server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/hrms-server .
# 监听 8080（容器内），环境变量由 compose 注入
EXPOSE 8080
CMD ["./hrms-server"]
