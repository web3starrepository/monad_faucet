FileName=bin/faucet

# 为 Windows 构建
echo "正在为 Windows 构建..."
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -ldflags="-w -s" -o ${FileName}.exe main.go

# 为 Linux 构建
echo "正在为 Linux 构建..."
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-gcc go build -ldflags="-linkmode external -extldflags -static" -o ${FileName}_linux main.go

# 为 macOS 构建
echo "正在为 macOS 构建..."
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o ${FileName}_mac main.go

echo "构建成功完成!"