# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is the Xbox Sing-box管理系统 (Xbox Sing-box Management System) - an enterprise-grade distributed sing-box node management platform built with Go. The system provides centralized management, configuration distribution, monitoring, and automated operations for large-scale sing-box node clusters.

## System Architecture

The system follows a distributed microservices architecture with two main components:

### Controller (Management Center)
- **Language**: Go 1.23+
- **Communication**: gRPC server + RESTful HTTP API
- **Database**: MySQL 8.0 with GORM
- **Monitoring**: Prometheus metrics exposure
- **Responsibilities**: Node registration, configuration management, monitoring aggregation, API services

### Agent (Node Proxy)
- **Language**: Go 1.23+ 
- **Communication**: gRPC client to Controller
- **Process Management**: sing-box binary lifecycle management
- **Monitoring**: System metrics collection and reporting
- **Responsibilities**: sing-box management, system monitoring, heartbeat reporting

### Supporting Services
- **MySQL**: Relational data storage (nodes, configs, rules)
- **Prometheus**: Metrics collection and storage
- **Grafana**: Monitoring dashboards and visualization
- **Redis**: Optional caching layer

## Common Development Commands

### Build and Development
```bash
# Install dependencies
make deps

# Generate protobuf code (required after proto changes)
make proto

# Build applications
make build

# Run tests
make test

# Clean build artifacts
make clean
```

### Local Development
```bash
# Run Controller locally
make run-controller

# Run Agent locally  
make run-agent

# Initialize database manually (if not using Docker)
make init-db
```

### Docker Deployment
```bash
# Full system installation (recommended for development)
./scripts/deploy.sh install

# Service management
./scripts/deploy.sh start      # Start all services
./scripts/deploy.sh stop       # Stop all services
./scripts/deploy.sh restart    # Restart all services
./scripts/deploy.sh status     # Check service health

# Monitoring and debugging
./scripts/deploy.sh logs [service_name]  # View logs
./scripts/deploy.sh backup               # Backup data
```

## Key Configuration Files

### Controller Configuration
- **Main config**: `configs/config.yaml`
- **Docker override**: Environment variables in `docker-compose.yml`
- **Database**: MySQL connection settings
- **gRPC**: Server binding configuration (default: 0.0.0.0:9090)
- **HTTP API**: REST API server settings (default: 0.0.0.0:8080)

### Agent Configuration  
- **Main config**: `configs/agent.yaml`
- **Controller address**: gRPC connection to Controller
- **sing-box config**: Path to sing-box configuration template
- **Heartbeat**: Reporting interval configuration

### sing-box Integration
- **Template**: `configs/sing-box.json` - Base configuration template
- **Management**: Agents dynamically manage sing-box process lifecycle
- **Configuration**: Hot-reload support via Controller API

## Development Architecture

### Project Structure
```
xbox/
├── cmd/                    # Application entry points
│   ├── controller/         # Controller main.go
│   └── agent/             # Agent main.go
├── internal/              # Private application code
│   ├── controller/        # Controller business logic
│   │   ├── grpc/         # gRPC service implementation
│   │   ├── service/      # Business service layer
│   │   └── repository/   # Data access layer
│   ├── agent/            # Agent business logic
│   │   ├── grpc/         # gRPC client implementation
│   │   ├── singbox/      # sing-box process management
│   │   └── monitor/      # System monitoring
│   ├── models/           # Data models and structs
│   ├── database/         # Database connection and migration
│   ├── config/           # Configuration management
│   └── monitoring/       # Prometheus metrics
├── proto/                # gRPC protocol definitions
├── api/                  # RESTful API handlers and routes
├── configs/              # Configuration templates
├── scripts/              # Deployment and utility scripts
└── monitoring/           # Prometheus and Grafana configs
```

### Communication Patterns
- **Controller ↔ Agent**: Bidirectional gRPC with streaming support
- **External ↔ Controller**: RESTful HTTP API
- **Controller ↔ Database**: GORM with MySQL
- **All Services ↔ Prometheus**: Metrics scraping endpoints

## API Integration Points

### Controller HTTP API (Port 8080)
- `GET /api/v1/health` - Health check endpoint
- `GET /api/v1/ready` - Readiness check  
- `GET /api/v1/agents` - List registered agents
- `POST /api/v1/configs` - Create/update configurations
- `PUT /api/v1/rules/{id}` - Update routing rules
- `GET /metrics` - Prometheus metrics endpoint

### Agent HTTP API (Port 8081)
- `GET /health` - Health check endpoint
- `GET /live` - Liveness probe
- `GET /metrics` - Prometheus metrics endpoint
- `GET /debug/runtime` - Runtime information
- `GET /debug/pprof/` - Performance profiling endpoints

### gRPC Services
- **AgentService**: Agent registration, heartbeat, configuration sync
- **Streaming**: Real-time configuration updates and status reporting

## Development Workflow

### Adding New Features
1. Define gRPC protobuf contracts in `proto/` if needed
2. Run `make proto` to generate Go code
3. Implement business logic in `internal/controller/` or `internal/agent/`
4. Add HTTP API endpoints in `api/` for external access
5. Update configuration schemas in `configs/`
6. Add monitoring metrics in `internal/monitoring/`
7. Write tests and run `make test`
8. Test with Docker deployment: `./scripts/deploy.sh install`

### Modifying sing-box Integration
1. Update sing-box template in `configs/sing-box.json`
2. Modify process management in `internal/agent/singbox/`
3. Test configuration hot-reload via Controller API
4. Verify sing-box proxy functionality on ports 1080 (SOCKS) and 8888 (HTTP)

### Database Schema Changes
1. Modify models in `internal/models/`
2. Update GORM auto-migration logic in `internal/database/`
3. Add migration scripts to `scripts/init.sql` if needed
4. Test with fresh database: `./scripts/deploy.sh cleanup && ./scripts/deploy.sh install`

## Monitoring and Debugging

### Health Checks
- **Controller**: `curl http://localhost:8080/api/v1/health`
- **Agent**: `curl http://localhost:8081/health`
- **Full status**: `./scripts/deploy.sh status`

### Metrics and Monitoring
- **Prometheus**: http://localhost:9000 - Metrics collection
- **Grafana**: http://localhost:3000 (admin/xbox123456) - Dashboards
- **Key metrics**: `xbox_agent_status`, `xbox_system_cpu_usage_percent`, `xbox_singbox_connections_active`

### Debugging
- **Runtime info**: `curl http://localhost:808{0,1}/debug/runtime`
- **Performance profiling**: `go tool pprof http://localhost:808{0,1}/debug/pprof/profile`
- **Container logs**: `docker-compose logs -f controller` / `docker-compose logs -f agent`

## Testing Strategy

### Unit Testing
```bash
# Run all tests
make test

# Test specific package
go test -v ./internal/controller/service/
go test -v ./internal/agent/singbox/
```

### Integration Testing
```bash
# Full system test with Docker
./scripts/deploy.sh install
# Verify all services are healthy
./scripts/deploy.sh status
# Test API endpoints
curl http://localhost:8080/api/v1/agents
```

### Load Testing
- Use the deployed system to test with multiple agent connections
- Monitor resource usage via Grafana dashboards
- Verify sing-box proxy performance under load

## Security Considerations

- **Network Isolation**: Services communicate via Docker internal network
- **Database Security**: MySQL with dedicated user credentials
- **API Access**: No authentication implemented yet (planned JWT integration)
- **Process Isolation**: sing-box runs in containerized environment
- **Monitoring Access**: Grafana requires authentication (admin/xbox123456)

## Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Language | Go 1.23+ | High-performance backend services |
| Communication | gRPC + HTTP | Service communication and external API |
| Database | MySQL 8.0 + GORM | Persistent data storage |
| Containerization | Docker + Compose | Application deployment |
| Monitoring | Prometheus + Grafana | Metrics and observability |
| Process Management | Native Go + sing-box | Proxy service management |
| Configuration | Viper + YAML | Application configuration |
| Logging | Logrus + Lumberjack | Structured logging with rotation |