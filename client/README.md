# OneDock Go Client

OneDock Go Client 是一个用于与 OneDock 容器编排服务进行交互的 Go 客户端库。

## 特性

- ✅ **完整的 API 支持**: 支持所有 OneDock REST API 功能
- ✅ **类型安全**: 强类型的请求和响应结构
- ✅ **认证支持**: 自动处理 Bearer Token 认证
- ✅ **错误处理**: 统一的错误类型和详细的错误信息
- ✅ **可配置**: 支持超时、调试模式等配置选项
- ✅ **标准库**: 仅使用 Go 标准库，无外部依赖

## 安装

```bash
go get github.com/aichy126/onedock/client
```

## 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/aichy126/onedock/client"
)

func main() {
    // 创建客户端
    onedockClient := client.New("http://localhost:8801", "your-token")

    // 或者使用配置选项
    onedockClient = client.New("http://localhost:8801", "your-token",
        client.WithTimeout(60*time.Second),
        client.WithDebug(true),
    )

    // 健康检查
    ping, err := onedockClient.Ping()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Server status: %s\n", ping.Message)
}
```

### 服务管理

#### 部署服务

```go
// 部署 nginx 服务
service, err := onedockClient.DeployService(&client.ServiceRequest{
    Name:         "nginx-web",
    Image:        "nginx",
    Tag:          "alpine",
    InternalPort: 80,
    PublicPort:   9203,
    Replicas:     3,
    Environment: map[string]string{
        "ENV": "production",
    },
})

if err != nil {
    log.Fatal(err)
}

fmt.Printf("Service deployed: %s, Status: %s\n", service.Name, service.Status)
```

#### 列出所有服务

```go
services, err := onedockClient.ListServices()
if err != nil {
    log.Fatal(err)
}

for _, service := range services {
    fmt.Printf("Service: %s, Status: %s, Replicas: %d\n",
        service.Name, service.Status, service.Replicas)
}
```

#### 获取服务详细状态

```go
status, err := onedockClient.GetServiceStatus("nginx-web")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Service: %s\n", status.Service.Name)
fmt.Printf("Total Replicas: %d\n", status.TotalReplicas)
fmt.Printf("Running Replicas: %d\n", status.RunningReplicas)
fmt.Printf("Access URL: %s\n", status.AccessURL)

// 查看实例详情
for i, instance := range status.Instances {
    fmt.Printf("Instance %d: %s (%s)\n", i+1, instance.ContainerName, instance.Status)
}
```

#### 扩缩容服务

```go
// 扩容到 5 个副本
err := onedockClient.ScaleService("nginx-web", 5)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Service scaled successfully")
```

#### 删除服务

```go
err := onedockClient.DeleteService("nginx-web")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Service deleted successfully")
```

### 高级功能

#### 带卷挂载的服务

```go
service, err := onedockClient.DeployService(&client.ServiceRequest{
    Name:         "mysql-db",
    Image:        "mysql",
    Tag:          "8.0",
    InternalPort: 3306,
    PublicPort:   9306,
    Replicas:     1,
    Environment: map[string]string{
        "MYSQL_ROOT_PASSWORD": "secret123",
        "MYSQL_DATABASE":      "myapp",
    },
    Volumes: []client.VolumeMount{
        {
            HostPath:      "/data/mysql",
            ContainerPath: "/var/lib/mysql",
            Mode:          "rw",
        },
    },
    WorkingDir: "/var/lib/mysql",
})
```

#### 代理统计信息

```go
stats, err := onedockClient.GetProxyStats()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Total Proxies: %d\n", stats.TotalProxies)
fmt.Printf("Load Balancers: %d\n", stats.LoadBalancers)

for _, proxy := range stats.ProxyDetails {
    fmt.Printf("Proxy on port %d: %s (%s)\n",
        proxy.PublicPort, proxy.ServiceName, proxy.ProxyType)
}
```

## 配置选项

### 客户端选项

```go
client := client.New("http://localhost:8801", "token",
    // 设置请求超时
    client.WithTimeout(60*time.Second),

    // 启用调试模式
    client.WithDebug(true),

    // 使用自定义 HTTP 客户端
    client.WithHTTPClient(&http.Client{
        Timeout: 30*time.Second,
        Transport: &http.Transport{
            MaxIdleConns: 10,
        },
    }),
)
```

### 认证

OneDock 支持多种认证方式，客户端会自动处理：

1. **Bearer Token**: `Authorization: Bearer <token>`
2. **Token Header**: `Token: <token>`
3. **Query 参数**: `?token=<token>`

```go
// 设置或更新 token
client.SetToken("new-token")

// 获取当前 token
currentToken := client.GetToken()
```

## 数据结构

### ServiceRequest

```go
type ServiceRequest struct {
    Name         string            `json:"name"`                // 服务名称
    Image        string            `json:"image"`               // Docker 镜像
    Tag          string            `json:"tag"`                 // 镜像标签
    InternalPort int               `json:"internal_port"`       // 容器内部端口
    Replicas     int               `json:"replicas,omitempty"`  // 副本数量
    Environment  map[string]string `json:"environment,omitempty"` // 环境变量
    EnvFile      string            `json:"env_file,omitempty"`    // 环境变量文件
    Volumes      []VolumeMount     `json:"volumes,omitempty"`     // 卷挂载
    Command      []string          `json:"command,omitempty"`     // 启动命令
    WorkingDir   string            `json:"working_dir,omitempty"` // 工作目录
    PublicPort   int               `json:"public_port,omitempty"` // 公共端口
}
```

### Service

```go
type Service struct {
    ID           string        `json:"id"`
    Name         string        `json:"name"`
    Image        string        `json:"image"`
    Tag          string        `json:"tag"`
    Status       ServiceStatus `json:"status"`
    PublicPort   int           `json:"public_port"`
    InternalPort int           `json:"internal_port"`
    Replicas     int           `json:"replicas"`
    CreatedAt    time.Time     `json:"created_at"`
    UpdatedAt    time.Time     `json:"updated_at"`
}
```

### ServiceStatusResponse

```go
type ServiceStatusResponse struct {
    Service         Service               `json:"service"`
    TotalReplicas   int                   `json:"total_replicas"`
    HealthyReplicas int                   `json:"healthy_replicas"`
    RunningReplicas int                   `json:"running_replicas"`
    StoppedReplicas int                   `json:"stopped_replicas"`
    FailedReplicas  int                   `json:"failed_replicas"`
    Instances       []ServiceInstanceInfo `json:"instances"`
    LoadBalancer    string                `json:"load_balancer"`
    AccessURL       string                `json:"access_url"`
    CreatedAt       time.Time             `json:"created_at"`
    UpdatedAt       time.Time             `json:"updated_at"`
}
```

## 完整示例

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/aichy126/onedock/client"
)

func main() {
    // 创建客户端
    onedockClient := client.New("http://localhost:8801", "your-token",
        client.WithTimeout(30*time.Second),
        client.WithDebug(false),
    )

    // 健康检查
    if _, err := onedockClient.Ping(); err != nil {
        log.Fatalf("Failed to connect to OneDock: %v", err)
    }

    serviceName := "nginx-demo"

    // 部署服务
    fmt.Println("Deploying service...")
    service, err := onedockClient.DeployService(&client.ServiceRequest{
        Name:         serviceName,
        Image:        "nginx",
        Tag:          "alpine",
        InternalPort: 80,
        PublicPort:   9203,
        Replicas:     2,
        Environment: map[string]string{
            "ENV": "demo",
        },
    })
    if err != nil {
        log.Fatalf("Failed to deploy service: %v", err)
    }

    fmt.Printf("Service deployed: %s (Status: %s)\n", service.Name, service.Status)

    // 等待一下让服务启动
    time.Sleep(5 * time.Second)

    // 获取服务状态
    fmt.Println("Getting service status...")
    status, err := onedockClient.GetServiceStatus(serviceName)
    if err != nil {
        log.Printf("Failed to get service status: %v", err)
    } else {
        fmt.Printf("Service status: %s, Running replicas: %d/%d\n",
            status.Service.Status, status.RunningReplicas, status.TotalReplicas)
        fmt.Printf("Access URL: %s\n", status.AccessURL)
    }

    // 扩容服务
    fmt.Println("Scaling service to 3 replicas...")
    if err := onedockClient.ScaleService(serviceName, 3); err != nil {
        log.Printf("Failed to scale service: %v", err)
    } else {
        fmt.Println("Service scaled successfully")
    }

    // 获取代理统计
    fmt.Println("Getting proxy stats...")
    stats, err := onedockClient.GetProxyStats()
    if err != nil {
        log.Printf("Failed to get proxy stats: %v", err)
    } else {
        fmt.Printf("Total proxies: %d, Load balancers: %d\n",
            stats.TotalProxies, stats.LoadBalancers)
    }

    // 列出所有服务
    fmt.Println("Listing all services...")
    services, err := onedockClient.ListServices()
    if err != nil {
        log.Printf("Failed to list services: %v", err)
    } else {
        for _, svc := range services {
            fmt.Printf("- %s: %s (%d replicas)\n", svc.Name, svc.Status, svc.Replicas)
        }
    }

    // 清理：删除服务
    fmt.Println("Cleaning up: deleting service...")
    if err := onedockClient.DeleteService(serviceName); err != nil {
        log.Printf("Failed to delete service: %v", err)
    } else {
        fmt.Println("Service deleted successfully")
    }
}
```
