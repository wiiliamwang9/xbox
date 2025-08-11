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
- **Process Management**: sing-box binary lifecycle management with automatic installation
- **Monitoring**: System metrics collection and reporting
- **Configuration**: Automatic sing-box configuration parsing and detailed logging
- **Responsibilities**: sing-box management, system monitoring, heartbeat reporting, configuration validation

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

### Docker Deployment (Recommended)
```bash
# Full system installation
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

### Agent Deployment
```bash
# Deploy agent to remote node (recommended for production)
./scripts/deploy_agent.sh <remote_ip> <ssh_password> [controller_ip]

# Test Agent startup flow with sing-box installation
./test_agent_startup.sh

# This script will:
# - Check and install sing-box automatically
# - Start Agent with configuration validation
# - Output detailed sing-box configuration info
# - Verify all startup processes
```

## Project Scripts

The project includes streamlined deployment and utility scripts:

### Core Scripts
- **`scripts/deploy.sh`** - Main Docker-based deployment script (install, start, stop, status)
- **`scripts/deploy_agent.sh`** - Remote agent deployment via SSH
- **`scripts/generate_proto.sh`** - Protocol buffer code generation

### Database Scripts
- **`scripts/init.sql`** - Main database initialization schema
- **`scripts/simple-init.sql`** - Simplified database schema
- **`scripts/add_ip_range_fields.sql`** - IP range management schema additions
- **`scripts/create_multiplex_table.sql`** - Multiplex configuration table schema

### Test Scripts (Root Level)
- **`test_agent_startup.sh`** - Agent startup and sing-box installation testing
- **`test_filter_functionality.sh`** - Protocol filter management testing
- **`test_multiplex_api.sh`** - Multiplex API functionality testing
- **`test_agent_multiplex_logs.sh`** - Agent multiplex logging verification
- **`test_agent_uninstall.sh`** - Agent uninstallation testing
- **`test_ip_range_functionality.sh`** - IP range management testing

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
├── scripts/              # Core deployment and utility scripts
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

### Controller HTTP API (Port 9000) - Filter Management
- `POST /api/v1/filter/blacklist` - Update protocol blacklist rules
- `POST /api/v1/filter/whitelist` - Update protocol whitelist rules
- `GET /api/v1/filter/config/{agent_id}` - Get agent filter configuration
- `GET /api/v1/filter/status/{agent_id}` - Get agent filter status
- `POST /api/v1/filter/rollback` - Rollback filter configuration

### Agent HTTP API (Port 8081)
- `GET /health` - Health check endpoint
- `GET /live` - Liveness probe
- `GET /metrics` - Prometheus metrics endpoint
- `GET /debug/runtime` - Runtime information
- `GET /debug/pprof/` - Performance profiling endpoints

### gRPC Services
- **AgentService**: Agent registration, heartbeat, configuration sync
- **FilterService**: Protocol-based blacklist/whitelist management
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

### Filter Functionality Testing
```bash
# Test protocol-based filter management
./test_filter_functionality.sh

# Test with specific agent ID
./test_filter_functionality.sh --agent-id your-agent-id

# Test with custom Controller URL
./test_filter_functionality.sh --url http://192.168.1.100:9000/api/v1
```

### Integration with SaaS Platform Testing
```bash
# Quick deployment test (recommended for initial testing)
./quick_deploy_test.sh

# Complete test suite (comprehensive testing)
./run_xbox_tests.sh

# Configuration management testing
./xbox_config_management_test.sh

# Full deployment test
./xbox_test_deployment.sh
```

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
| Web Framework | Gin + Gorilla Mux | HTTP API and routing |

## Filter Management System

### Protocol-based Filtering
The system supports advanced protocol-level filtering with blacklist/whitelist capabilities:

- **Supported Protocols**: HTTP, HTTPS, SOCKS5, Shadowsocks, VMess, Trojan, VLESS
- **Filter Types**: Domain, IP address, and port-based filtering
- **Operations**: Add, remove, replace, clear operations for dynamic rule management
- **Configuration**: JSON-based configuration with version control and rollback support
- **Integration**: Automatic sing-box configuration generation and hot-reload

## Multiplex Configuration System

### Connection Limit Management
The system provides advanced multiplex configuration management for optimizing proxy connection performance:

- **Supported Protocols**: VMess, VLESS, Trojan, Shadowsocks
- **Key Features**:
  - Dynamic connection limit configuration via REST API
  - Real-time configuration updates without service restart
  - Database persistence with version control
  - Batch configuration operations
  - Statistical monitoring and reporting
- **Configuration Parameters**:
  - `max_connections`: Maximum connection count (1-32)
  - `min_streams`: Minimum stream count (1-32) 
  - `padding`: Enable/disable padding
  - `brutal`: Congestion control configuration (optional)
- **Constraints**: Only `max_connections` is configured (as per sing-box best practices)
- **Prerequisites**: Multiplex must be enabled before applying connection limits

### Agent Configuration Logging
The Agent provides comprehensive logging for multiplex configuration updates:

- **Detailed Request Logging**: Full request parameters including protocol, connection limits, and Brutal settings
- **Configuration Comparison**: Shows original vs. new configuration for each outbound
- **Step-by-Step Process**: Logs each stage of the configuration update process
- **Performance Metrics**: Records operation timing and success/failure status
- **Sing-box Integration**: Detailed logging of sing-box configuration application

#### Example Agent Log Output
```
=== 收到多路复用配置更新请求 ===
Agent ID: test-agent-001
Protocol: vmess
配置详情:
  启用状态: true
  协议: smux
  最大连接数: 8
  最小流数: 4
  填充: false
  Brutal配置: map[down:200 Mbps up:100 Mbps]
正在更新sing-box多路复用配置...
找到匹配的出站配置: Tag=vmess-out, Type=vmess
  原配置: 未配置
  新配置: enabled=true, max_conn=8, min_streams=4, padding=false
  设置Brutal上传带宽: 100 Mbps
  设置Brutal下载带宽: 200 Mbps
多路复用配置更新完成，影响的出站: [vmess-out]
正在应用多路复用配置到sing-box...
sing-box多路复用配置应用成功
多路复用配置更新成功 (耗时: 45.2ms)
=== 多路复用配置更新请求处理完成 ===
```

### Multiplex Configuration Commands
```bash
# Test multiplex API functionality
./test_multiplex_api.sh

# Test Agent multiplex logs and configuration updates
./test_agent_multiplex_logs.sh

# Update multiplex configuration via API
curl -X POST -H "Content-Type: application/json" \
  -d '{"agent_id": "agent-id", "protocol": "vmess", "enabled": true, "max_connections": 8}' \
  http://localhost:9000/api/v1/multiplex/config

# Get multiplex configuration
curl -X GET http://localhost:9000/api/v1/multiplex/config/agent-id?protocol=vmess

# Get multiplex statistics
curl -X GET http://localhost:9000/api/v1/multiplex/status

# Monitor Agent logs for configuration updates
tail -f logs/xbox/agents/agent.log | grep -i multiplex
```

### Filter Management Commands
```bash
# Test filter functionality
./test_filter_functionality.sh

# Add blacklist rules via API
curl -X POST -H "Content-Type: application/json" \
  -d '{"agent_id": "agent-id", "protocol": "http", "domains": ["blocked.site"], "operation": "add"}' \
  http://localhost:9000/api/v1/filter/blacklist

# Get filter configuration
curl -X GET http://localhost:9000/api/v1/filter/config/agent-id

# Rollback configuration
curl -X POST -H "Content-Type: application/json" \
  -d '{"agent_id": "agent-id", "reason": "rollback test"}' \
  http://localhost:9000/api/v1/filter/rollback
```

## Integration Testing Framework

The system includes comprehensive integration testing capabilities:

### Test Coverage
- **Agent Deployment**: SSH-based automated deployment to remote nodes
- **Database Synchronization**: Node information sync between Xbox Controller and SaaS Backend
- **sing-box Integration**: Multi-protocol proxy service deployment and configuration
- **System Monitoring**: CPU, memory, network monitoring and heartbeat functionality
- **Configuration Management**: Dynamic configuration updates including blacklist/whitelist rules
- **API Integration**: REST API testing between SaaS Backend and Xbox Controller
- **Protocol Support**: Verification of SOCKS, HTTP, Shadowsocks, VMess, Trojan, VLESS protocols

### Test Target Configuration
- **Target Node**: 165.254.16.244 (configurable)
- **SSH Authentication**: Password-based (automated via sshpass)
- **Dependencies**: Automatically installs required tools (sshpass, curl, jq)

## Enhanced Agent Startup Process

The Agent has been enhanced with automatic sing-box installation and detailed configuration reporting:

### Startup Flow
1. **Environment Check**: Validates Go environment and system tools
2. **sing-box Installation**: Automatically detects, downloads, and installs sing-box if not present
3. **Configuration Validation**: Parses and validates sing-box configuration files
4. **Detailed Logging**: Outputs comprehensive configuration information including:
   - Log configuration (level, output, timestamp)
   - DNS settings (servers, rules, FakeIP)
   - Inbound configurations (protocols, ports, TLS, users)
   - Outbound configurations (protocols, servers, encryption)
   - Route rules (domains, IPs, GeoIP, Geosite)
   - Experimental features (Clash API, V2Ray API, cache files)
   - NTP configuration

### sing-box Auto Installation
- **Detection**: Checks both local binary path and system PATH
- **Download Methods**: 
  - Primary: Official installation script from `https://sing-box.app/install.sh`
  - Fallback: Direct GitHub releases download with architecture detection
- **Supported Platforms**: Linux (amd64, arm64, 386, armv7), Windows, macOS
- **Version Reporting**: Automatically detects and reports installed version

### Configuration Structure Support
The Agent now supports complete sing-box configuration parsing with all field types:
- **Inbounds**: Mixed, HTTP, SOCKS, Shadowsocks, VMess, Trojan, VLESS, Hysteria
- **Outbounds**: Direct, Block, DNS, all proxy protocols with full configuration options
- **Advanced Features**: TLS/Reality, Transport layers, Multiplex, ECH, uTLS
- **Route Rules**: Full routing rule support with GeoIP/Geosite integration
- **Experimental**: Clash API, V2Ray API, Cache files, Debug options

### Testing and Validation
Use the provided test script to validate the enhanced startup process:

```bash
./test_agent_startup.sh
```

This comprehensive test will verify:
- Environment dependencies
- Project building
- Agent startup with sing-box auto-installation  
- Configuration parsing and detailed output
- Process management and cleanup