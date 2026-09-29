.PHONY: run build tidy vet test clean

# 直接运行（需要本地 MongoDB 27017）
run:
	go run ./cmd/server

# 构建产物到 bin/
build:
	go build -ldflags="-s -w" -o bin/serveradmin.exe ./cmd/server

tidy:
	go mod tidy

vet:
	go vet ./...

test:
	go test ./...

# 构建跨平台 Linux 版本（部署服务器用）
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/serveradmin-linux ./cmd/server

clean:
	rm -rf bin/
