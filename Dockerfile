# ServerAdmin 后台管理系统多阶段构建
FROM golang:1.27-alpine AS builder
WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/serveradmin ./cmd/server

FROM alpine:3.20
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=builder /out/serveradmin /app/serveradmin
COPY config/config.yaml /app/config.yaml
USER app
EXPOSE 8080
VOLUME ["/app/uploads", "/app/logs"]
ENTRYPOINT ["/app/serveradmin"]
