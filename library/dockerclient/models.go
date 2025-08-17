package dockerclient

import "github.com/docker/docker/client"

// Service 服务配置结构体，用于Docker操作
type Service struct {
	Name         string            // 服务名称
	Image        string            // Docker镜像名称
	Tag          string            // 镜像标签
	PublicPort   int               // 公共端口（用户访问端口）
	InternalPort int               // 容器内部端口
	DockerPort   int               // Docker映射端口（动态分配）
	Environment  map[string]string // 环境变量
	EnvFile      string            // 环境变量文件路径
	Volumes      []VolumeMount     // 卷挂载配置
	Command      []string          // 启动命令
	WorkingDir   string            // 工作目录
	Replicas     int               // 副本数量
}

// VolumeMount 卷挂载结构体
type VolumeMount struct {
	Source      string // 主机路径
	Destination string // 容器内路径
	ReadOnly    bool   // 是否只读挂载
}

// ContainerNameInfo 容器名称解析结果
type ContainerNameInfo struct {
	ServiceName   string // 服务名称
	PublicPort    int    // 公共端口（用户访问端口）
	ContainerPort int    // 容器端口（Docker映射端口）
	ReplicaIndex  int    // 副本索引
}

// DockerClient Docker客户端结构体
type DockerClient struct {
	cli               client.APIClient // Docker API客户端
	containerPrefix   string           // 容器名称前缀
	internalPortStart int              // 内部端口起始
}

// ContainerInfo 容器信息结构体
type ContainerInfo struct {
	ID        string            // 容器ID
	Name      string            // 容器名称
	Image     string            // 镜像名称
	Status    string            // 容器状态
	Ports     []PortMapping     // 端口映射
	Labels    map[string]string // 标签
	State     string            // 运行状态
	CreatedAt string            // 创建时间
}

// PortMapping 端口映射信息结构体
type PortMapping struct {
	HostPort      string // 主机端口
	ContainerPort string // 容器端口
	Protocol      string // 协议类型
}
