# OneDock

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org)
[![Build Status](https://github.com/aichy126/onedock/workflows/CI/badge.svg)](https://github.com/aichy126/onedock/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/aichy126/onedock)](https://goreportcard.com/report/github.com/aichy126/onedock)

[中文文档](README_zh.md) | English

A powerful Docker container orchestration service built with Go and Gin framework, providing RESTful APIs for deploying, managing, and scaling containerized services with intelligent port proxying and load balancing.

## ✨ Features

- **🚀 Container Lifecycle Management**: Deploy, update, delete, and scale containerized services
- **🔄 Smart Port Management**: Automatic port allocation starting from configured base port
- **⚖️ Intelligent Load Balancing**: Automatically switch between single-replica proxy and load balancer based on replica count
- **📦 Cache Optimization**: Port mapping cache with TTL and manual cleanup support
- **🔧 Service Recovery**: Auto-recovery of port proxy services on startup
- **📊 Health Monitoring**: Container status monitoring and instance details query
- **📖 Swagger Documentation**: Complete API documentation with online testing support
- **🔀 Multiple Load Balancing Strategies**: Round-robin, least connections, and weighted strategies

## 🏗️ Architecture

OneDock adopts a layered architecture with the following core components:

- **API Layer** (`/api/`): RESTful route definitions and HTTP handlers
- **Service Layer** (`/service/`): Core service management, Docker integration, and port management
- **Model Layer** (`/models/`): Complete service data structures and API models
- **Docker Client** (`/library/dockerclient/`): Docker operations abstraction layer
- **Cache Layer** (`/library/cache/`): Memory and Redis cache implementations
- **Utilities** (`/utils/`): Configuration management and common utilities

### Intelligent Proxy System

OneDock features an intelligent proxy system that automatically chooses the optimal proxying strategy:

- **Single Replica Mode**: Uses `httputil.ReverseProxy` for direct proxying when `replicas = 1`
- **Load Balancer Mode**: Automatically enables `LoadBalancer` when `replicas > 1`
- **Dynamic Switching**: Seamlessly switches between modes during scaling operations
- **Access Consistency**: External access port remains unchanged regardless of replica count

## 🚀 Quick Start

### Prerequisites

- Go 1.24 or higher
- Docker (must be accessible from the host system)
- Git

### ⚠️ Important Note on Deployment

**OneDock should be deployed as a native binary on the host system, NOT as a Docker container.**

Since OneDock is a Docker container orchestration service that needs to manage Docker containers, running it inside a Docker container would create unnecessary complexity and potential issues:

- **Docker-in-Docker (DinD) complexity**: Requires complex volume mounts and privileged containers
- **Network conflicts**: Port management and proxy functionality may conflict with container networking
- **Security concerns**: Requires elevated privileges and Docker socket access
- **Resource overhead**: Additional layer of containerization without benefits

### Recommended Deployment Methods

1. **Direct Binary Deployment** (Recommended)
2. **Systemd Service** (For production environments)
3. **Process Manager** (PM2, Supervisor, etc.)

### Installation

📖 **For detailed deployment instructions, see [Deploy Guide](./deploy/README.md)**

#### Quick Installation

```bash
# 1. Clone the repository
git clone https://github.com/aichy126/onedock.git
cd onedock

# 2. Build the binary
go build -o onedock

# 3. Install as systemd service (Linux)
sudo ./deploy/install.sh
```

#### Manual Build and Run

1. **Clone the repository**
   ```bash
   git clone https://github.com/aichy126/onedock.git
   cd onedock
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Configure the application**
   ```bash
   cp config.toml.example config.toml
   # Edit config.toml according to your environment
   ```

4. **Run the development server**
   ```bash
   ./dev.sh
   ```
   Or run directly:
   ```bash
   go run main.go
   ```

5. **Access the API**
   - API Base URL: `http://localhost:8801`
   - Swagger UI: `http://localhost:8801/swagger/index.html`

### Build for Production

```bash
# Build binary
go build -o onedock

# Cross-compile for Linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o onedock-linux

# Generate Swagger docs
swag init
```

## 📖 API Documentation

### Service Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/onedock/` | Deploy or update service |
| `GET` | `/onedock/` | List all services |
| `GET` | `/onedock/:name` | Get specific service details |
| `DELETE` | `/onedock/:name` | Delete service |

### Service Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/onedock/:name/status` | Get detailed service status |
| `POST` | `/onedock/:name/scale` | Scale service replicas |

### Monitoring

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/onedock/ping` | Health check and debug info |
| `GET` | `/onedock/proxy/stats` | Get port proxy statistics |

## 💡 Usage Examples

### Deploy a Service

```bash
curl -X 'POST' 'http://127.0.0.1:8801/onedock' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "nginx-web",
    "image": "nginx",
    "tag": "alpine",
    "internal_port": 80,
    "public_port": 9203,
    "replicas": 3,
    "environment": {
      "ENV": "production"
    }
  }'
```

### Scale a Service

```bash
curl -X 'POST' 'http://127.0.0.1:8801/onedock/nginx-web/scale' \
  -H 'Content-Type: application/json' \
  -d '{"replicas": 5}'
```

### Get Service Status

```bash
curl http://127.0.0.1:8801/onedock/nginx-web/status
```

### Access the Service

```bash
curl http://localhost:9203/
# Requests are automatically load-balanced across containers
```

## ⚙️ Configuration

Edit `config.toml` to customize your deployment:

```toml
[local]
address = ":8801"        # Service listen address
debug = true             # Gin debug mode

[swaggerui]
show = true              # Show Swagger UI
protocol = "http"        # Protocol
host = "127.0.0.1"      # Host address
address = ":8801"        # Port

[container]
prefix = "onedock"                    # Container name prefix
internal_port_start = 30000          # Internal port start value
cache_ttl = 300                      # Cache expiration time (seconds)
load_balance_strategy = "round_robin" # Load balancing strategy
```

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run specific module tests
go test ./service/
go test ./library/cache/
go test ./library/dockerclient/

# Run tests with coverage
go test -cover ./...
```


## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Gin](https://github.com/gin-gonic/gin) - HTTP web framework
- [Docker](https://www.docker.com/) - Containerization platform
- [Swagger](https://swagger.io/) - API documentation

## 📞 Support

- 🐛 Issues: [GitHub Issues](https://github.com/aichy126/onedock/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/aichy126/onedock/discussions)

---
