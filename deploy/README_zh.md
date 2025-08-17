# OneDock 部署指南

[English](README.md) | 中文文档

## 🚀 快速安装（推荐）

### 自动安装脚本

使用提供的安装脚本可以快速部署 OneDock 作为 systemd 服务：

```bash
# 1. 构建二进制文件
go build -o onedock

# 2. 运行安装脚本（需要 root 权限）
sudo ./deploy/install.sh
```

安装脚本会自动：
- 创建 `onedock` 用户和组
- 安装二进制文件到 `/opt/onedock/`
- 配置 systemd 服务
- 启动服务

### 验证安装

```bash
# 检查服务状态
sudo systemctl status onedock

# 查看日志
sudo journalctl -u onedock -f

# 测试 API
curl http://localhost:8801/onedock/ping
```

## 📋 手动安装

如果您想手动控制安装过程：

### 1. 准备环境

```bash
# 创建用户和组
sudo groupadd -r onedock
sudo useradd -r -g onedock -d /opt/onedock -s /bin/false onedock

# 将用户添加到 docker 组
sudo usermod -aG docker onedock

# 创建安装目录
sudo mkdir -p /opt/onedock/logs
sudo chown -R onedock:onedock /opt/onedock
```

### 2. 安装二进制文件

```bash
# 构建
go build -o onedock

# 安装
sudo cp onedock /opt/onedock/
sudo cp config.toml.example /opt/onedock/config.toml
sudo chown onedock:onedock /opt/onedock/onedock
sudo chmod 755 /opt/onedock/onedock
```

### 3. 配置 systemd 服务

```bash
# 复制服务文件
sudo cp deploy/onedock.service /etc/systemd/system/

# 重新加载 systemd
sudo systemctl daemon-reload

# 启用并启动服务
sudo systemctl enable onedock
sudo systemctl start onedock
```

## 🔧 配置

### 主要配置文件

编辑 `/opt/onedock/config.toml`：

```toml
[local]
address = ":8801"        # 服务监听地址
debug = false            # 生产环境建议关闭调试模式

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

### 服务管理命令

```bash
# 启动服务
sudo systemctl start onedock

# 停止服务
sudo systemctl stop onedock

# 重启服务
sudo systemctl restart onedock

# 查看服务状态
sudo systemctl status onedock

# 查看实时日志
sudo journalctl -u onedock -f

# 查看最近日志
sudo journalctl -u onedock -n 50
```

## 🔒 安全考虑

### 用户权限

- OneDock 服务运行在专用的 `onedock` 用户下
- 该用户被添加到 `docker` 组以访问 Docker daemon
- 服务使用受限的权限运行

### 网络安全

- 默认只监听本地地址（127.0.0.1）
- 如需外部访问，请谨慎配置防火墙规则
- 考虑使用反向代理（如 Nginx）添加 SSL/TLS

### 建议的防火墙规则

```bash
# 只允许特定 IP 访问（示例）
sudo ufw allow from 192.168.1.0/24 to any port 8801

# 或者使用 iptables
sudo iptables -A INPUT -p tcp --dport 8801 -s 192.168.1.0/24 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8801 -j DROP
```

## 🔄 升级

### 停止服务并升级

```bash
# 停止服务
sudo systemctl stop onedock

# 备份当前版本
sudo cp /opt/onedock/onedock /opt/onedock/onedock.backup

# 构建新版本
go build -o onedock

# 安装新版本
sudo cp onedock /opt/onedock/
sudo chown onedock:onedock /opt/onedock/onedock

# 启动服务
sudo systemctl start onedock
```

### 零停机升级（推荐）

```bash
# 构建新版本
go build -o onedock-new

# 替换二进制文件
sudo cp onedock-new /opt/onedock/onedock
sudo chown onedock:onedock /opt/onedock/onedock

# 重启服务
sudo systemctl restart onedock
```

## 🗑️ 卸载

```bash
# 停止并禁用服务
sudo systemctl stop onedock
sudo systemctl disable onedock

# 删除服务文件
sudo rm /etc/systemd/system/onedock.service
sudo systemctl daemon-reload

# 删除安装目录
sudo rm -rf /opt/onedock

# 删除用户和组（可选）
sudo userdel onedock
sudo groupdel onedock
```

## 🐛 故障排查

### 常见问题

1. **服务启动失败**
   ```bash
   # 查看详细错误信息
   sudo journalctl -u onedock -f
   ```

2. **Docker 权限问题**
   ```bash
   # 确认用户在 docker 组中
   groups onedock
   
   # 重新添加到 docker 组
   sudo usermod -aG docker onedock
   ```

3. **端口冲突**
   ```bash
   # 检查端口占用
   sudo netstat -tlnp | grep 8801
   
   # 修改配置文件中的端口
   sudo vim /opt/onedock/config.toml
   ```

4. **配置文件问题**
   ```bash
   # 验证配置文件格式
   /opt/onedock/onedock --config /opt/onedock/config.toml --validate
   ```

### 日志级别

修改配置文件中的日志级别：

```toml
[local]
debug = true  # 启用调试模式获取更多日志
```

### 健康检查

```bash
# API 健康检查
curl -f http://localhost:8801/onedock/ping || echo "Service is down"

# 服务状态检查
systemctl is-active onedock
```