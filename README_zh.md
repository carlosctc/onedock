# OneDock

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org)
[![Build Status](https://github.com/aichy126/onedock/workflows/CI/badge.svg)](https://github.com/aichy126/onedock/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/aichy126/onedock)](https://goreportcard.com/report/github.com/aichy126/onedock)

中文文档 | [English](README.md)

OneDock 是一个基于 Go 和 Gin 框架构建的强大 Docker 容器编排服务，提供 RESTful API 用于部署、管理和扩展容器化服务，支持智能端口代理和负载均衡。

## ✨ 特性

- **🚀 容器生命周期管理**: 部署、更新、删除和扩缩容容器化服务
- **🔄 智能端口管理**: 从配置的基础端口开始自动分配容器端口
- **⚖️ 智能负载均衡**: 根据副本数自动切换单副本代理或负载均衡器
- **📦 缓存优化**: 端口映射缓存，支持 TTL 和手动清理
- **🔧 服务恢复**: 启动时自动恢复端口代理服务
- **📊 健康监控**: 容器状态监控和实例详情查询
- **📖 Swagger 文档**: 完整的 API 文档，支持在线测试
- **🔀 多种负载均衡策略**: 轮询、最少连接数和权重策略

## 🏗️ 架构

OneDock 采用分层架构，包含以下核心组件：

- **API 层** (`/api/`): RESTful 路由定义和 HTTP 处理器
- **服务层** (`/service/`): 核心服务管理、Docker 集成和端口管理
- **模型层** (`/models/`): 完整的服务数据结构和 API 模型
- **Docker 客户端** (`/library/dockerclient/`): Docker 操作抽象层
- **缓存层** (`/library/cache/`): 内存和 Redis 缓存实现
- **工具层** (`/utils/`): 配置管理和通用工具

### 智能代理系统

OneDock 具有智能代理系统，能够自动选择最优的代理策略：

- **单副本模式**: 当 `replicas = 1` 时，使用 `httputil.ReverseProxy` 直接代理
- **负载均衡模式**: 当 `replicas > 1` 时，自动启用 `LoadBalancer`
- **动态切换**: 扩缩容时无缝切换代理模式
- **访问一致性**: 无论副本数如何，外部访问端口保持不变

## 🚀 快速开始

### 前置条件

- Go 1.24 或更高版本
- Docker（必须可从宿主系统访问）
- Git

### ⚠️ 重要的部署说明

**OneDock 应该作为原生二进制文件部署在宿主系统上，而不是作为 Docker 容器运行。**

由于 OneDock 是一个需要管理 Docker 容器的容器编排服务，在 Docker 容器内运行会产生不必要的复杂性和潜在问题：

- **Docker-in-Docker (DinD) 复杂性**: 需要复杂的卷挂载和特权容器
- **网络冲突**: 端口管理和代理功能可能与容器网络产生冲突
- **安全问题**: 需要提升权限和 Docker socket 访问
- **资源开销**: 额外的容器化层没有带来好处

### 推荐的部署方法

1. **直接二进制部署**（推荐）
2. **Systemd 服务**（生产环境）
3. **进程管理器**（PM2、Supervisor 等）

### 安装

📖 **详细部署说明请参考 [部署指南](./deploy/README_zh.md)**

#### 快速安装

```bash
# 1. 克隆仓库
git clone https://github.com/aichy126/onedock.git
cd onedock

# 2. 构建二进制文件
go build -o onedock

# 3. 安装为 systemd 服务（Linux）
sudo ./deploy/install.sh
```

#### 手动构建和运行

1. **克隆仓库**
   ```bash
   git clone https://github.com/aichy126/onedock.git
   cd onedock
   ```

2. **安装依赖**
   ```bash
   go mod tidy
   ```

3. **配置应用**
   ```bash
   cp config.toml.example config.toml
   # 根据你的环境编辑 config.toml
   ```

4. **运行开发服务器**
   ```bash
   ./dev.sh
   ```
   或直接运行：
   ```bash
   go run main.go
   ```

5. **访问 API**
   - API 基础地址: `http://localhost:8801`
   - Swagger UI: `http://localhost:8801/swagger/index.html`

### 生产环境构建

```bash
# 构建二进制文件
go build -o onedock

# 交叉编译为 Linux 版本
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o onedock-linux

# 生成 Swagger 文档
swag init
```

## 📖 API 文档

### 服务管理

| 方法 | 端点 | 描述 |
|------|------|------|
| `POST` | `/onedock/` | 部署或更新服务 |
| `GET` | `/onedock/` | 列出所有服务 |
| `GET` | `/onedock/:name` | 获取特定服务详情 |
| `DELETE` | `/onedock/:name` | 删除服务 |

### 服务操作

| 方法 | 端点 | 描述 |
|------|------|------|
| `GET` | `/onedock/:name/status` | 获取详细服务状态 |
| `POST` | `/onedock/:name/scale` | 扩缩容服务副本 |

### 监控

| 方法 | 端点 | 描述 |
|------|------|------|
| `GET` | `/onedock/ping` | 健康检查和调试信息 |
| `GET` | `/onedock/proxy/stats` | 获取端口代理统计 |

## 💡 使用示例

### 部署服务

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

### 扩缩容服务

```bash
curl -X 'POST' 'http://127.0.0.1:8801/onedock/nginx-web/scale' \
  -H 'Content-Type: application/json' \
  -d '{"replicas": 5}'
```

### 获取服务状态

```bash
curl http://127.0.0.1:8801/onedock/nginx-web/status
```

### 访问服务

```bash
curl http://localhost:9203/
# 请求自动在容器间负载均衡
```

## ⚙️ 配置

编辑 `config.toml` 来自定义你的部署：

```toml
[local]
address = ":8801"        # 服务监听地址
debug = true             # Gin 调试模式

[swaggerui]
show = true              # 是否显示 Swagger UI
protocol = "http"        # 协议
host = "127.0.0.1"      # 主机地址
address = ":8801"        # 端口

[container]
prefix = "onedock"                    # 容器名称前缀
internal_port_start = 30000          # 内部端口起始值
cache_ttl = 300                      # 缓存过期时间（秒）
load_balance_strategy = "round_robin" # 负载均衡策略
```

## 🧪 测试

```bash
# 运行所有测试
go test ./...

# 运行特定模块测试
go test ./service/
go test ./library/cache/
go test ./library/dockerclient/

# 运行测试并显示覆盖率
go test -cover ./...
```

## 🤝 贡献

我们欢迎贡献！请查看我们的[贡献指南](CONTRIBUTING.md)了解详情。

1. Fork 此仓库
2. 创建你的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交你的更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 📄 许可证

此项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [Gin](https://github.com/gin-gonic/gin) - HTTP web 框架
- [Docker](https://www.docker.com/) - 容器化平台
- [Swagger](https://swagger.io/) - API 文档

## 📞 支持

- 🐛 问题: [GitHub Issues](https://github.com/aichy126/onedock/issues)
- 💬 讨论: [GitHub Discussions](https://github.com/aichy126/onedock/discussions)

---

