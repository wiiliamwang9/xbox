.PHONY: proto build clean test

# 生成protobuf代码
proto:
	@echo "Generating protobuf code..."
	@./scripts/generate_proto.sh

# 构建应用
build: proto
	@echo "Building applications..."
	@go build -o bin/controller ./cmd/controller
	@go build -o bin/agent ./cmd/agent

# 清理构建文件
clean:
	@echo "Cleaning build files..."
	@rm -rf bin/
	@rm -f proto/agent/*.pb.go

# 运行测试
test:
	@echo "Running tests..."
	@go test -v ./...

# 安装依赖
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# 初始化数据库
init-db:
	@echo "Initializing database..."
	@mysql -u root -p < scripts/init.sql

# 运行controller
run-controller: build
	@echo "Starting controller..."
	@./bin/controller

# 运行agent
run-agent: build
	@echo "Starting agent..."
	@./bin/agent