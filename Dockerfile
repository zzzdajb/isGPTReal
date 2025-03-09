# 第一阶段：构建Go应用
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的构建工具
RUN apk add --no-cache git

# 复制Go模块文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用程序
RUN CGO_ENABLED=0 GOOS=linux go build -o isGPTReal ./cmd/main.go

# 第二阶段：创建最终镜像
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates

# 设置工作目录
WORKDIR /app

# 从构建阶段复制编译好的二进制文件
COPY --from=builder /app/isGPTReal .

# 复制前端资源
COPY --from=builder /app/static ./static
COPY --from=builder /app/templates ./templates

# 暴露应用端口（默认8080）
EXPOSE 8080

# 设置环境变量
ENV GIN_MODE=release

# 运行命令
ENTRYPOINT ["./isGPTReal"]
# 可以通过命令行参数覆盖默认设置
# 例如: docker run -e OPENAI_API_KEY=your_key -e OPENAI_ENDPOINT=your_endpoint -p 8080:8080 isgptreal 