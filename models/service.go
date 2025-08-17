package models

import (
	"time"

	"github.com/aichy126/onedock/library/dockerclient"
)

// ServiceStatus 服务状态
type ServiceStatus string

const (
	StatusStopped  ServiceStatus = "stopped"
	StatusStarting ServiceStatus = "starting"
	StatusRunning  ServiceStatus = "running"
	StatusStopping ServiceStatus = "stopping"
	StatusFailed   ServiceStatus = "failed"
	StatusUpdating ServiceStatus = "updating"
)

// 复用dockerclient的数据结构
type VolumeMount = dockerclient.VolumeMount
type ContainerInfo = dockerclient.ContainerInfo
type PortMapping = dockerclient.PortMapping

// Service API响应用的服务信息
type Service struct {
	ID           string        `json:"id" example:"svc_1234567890" description:"服务唯一标识"`
	Name         string        `json:"name" example:"nginx-web" description:"服务名称"`
	Image        string        `json:"image" example:"nginx" description:"Docker镜像名称"`
	Tag          string        `json:"tag" example:"alpine" description:"镜像标签"`
	Status       ServiceStatus `json:"status" example:"running" description:"服务运行状态"`
	PublicPort   int           `json:"public_port" example:"30000" description:"对外暴露端口"`
	InternalPort int           `json:"internal_port" example:"80" description:"容器内部端口"`
	Replicas     int           `json:"replicas" example:"3" description:"实际运行的副本数量"`
	CreatedAt    time.Time     `json:"created_at" example:"2023-01-01T00:00:00Z" description:"创建时间"`
	UpdatedAt    time.Time     `json:"updated_at" example:"2023-01-01T00:00:00Z" description:"更新时间"`
}

// ServiceRequest 直接使用dockerclient.Service结构（继承并添加JSON标签）
type ServiceRequest struct {
	Name         string            `json:"name" binding:"required" example:"nginx-web" description:"服务名称"`
	Image        string            `json:"image" binding:"required" example:"nginx" description:"Docker镜像名称"`
	Tag          string            `json:"tag" binding:"required" example:"alpine" description:"镜像标签"`
	InternalPort int               `json:"internal_port" binding:"required" example:"80" description:"容器内部端口"`
	Replicas     int               `json:"replicas" example:"1" description:"副本数量"`
	Environment  map[string]string `json:"environment" description:"环境变量"`
	EnvFile      string            `json:"env_file" description:"环境变量文件路径"`
	Volumes      []VolumeMount     `json:"volumes" description:"卷挂载配置"`
	Command      []string          `json:"command" description:"启动命令覆盖"`
	WorkingDir   string            `json:"working_dir" example:"/app" description:"工作目录"`
	PublicPort   int               `json:"public_port,omitempty" example:"30000" description:"可选的对外暴露端口，不填则自动分配"`
}

// ScaleRequest 扩缩容请求
// @Description 服务扩缩容请求参数
type ScaleRequest struct {
	Replicas int `json:"replicas" binding:"required" example:"3" description:"目标副本数量"`
}

// ServiceInstanceInfo 服务实例详细信息
type ServiceInstanceInfo struct {
	ID            string            `json:"id" example:"inst_1234567890" description:"实例唯一标识"`
	ContainerID   string            `json:"container_id" example:"abc123def456" description:"Docker容器ID"`
	ContainerName string            `json:"container_name" example:"onedock-nginx-1" description:"容器名称"`
	ServiceName   string            `json:"service_name" example:"nginx-web" description:"所属服务名称"`
	Status        string            `json:"status" example:"running" description:"容器运行状态"`
	HealthStatus  string            `json:"health_status" example:"healthy" description:"健康检查状态"`
	PublicPort    int               `json:"public_port" example:"30000" description:"对外访问端口"`
	ContainerPort int               `json:"container_port" example:"30001" description:"容器映射端口"`
	InternalPort  int               `json:"internal_port" example:"80" description:"容器内部端口"`
	Image         string            `json:"image" example:"nginx:alpine" description:"镜像名称"`
	CreatedAt     time.Time         `json:"created_at" example:"2023-01-01T00:00:00Z" description:"创建时间"`
	StartedAt     time.Time         `json:"started_at" example:"2023-01-01T00:00:00Z" description:"启动时间"`
	IPAddress     string            `json:"ip_address" example:"172.17.0.2" description:"容器IP地址"`
	Labels        map[string]string `json:"labels" description:"容器标签"`
	RestartCount  int               `json:"restart_count" example:"0" description:"重启次数"`
	Uptime        string            `json:"uptime" example:"2h30m" description:"运行时长"`
	CPUUsage      float64           `json:"cpu_usage" example:"0.5" description:"CPU使用率"`
	MemoryUsage   float64           `json:"memory_usage" example:"64.5" description:"内存使用(MB)"`
	MemoryLimit   float64           `json:"memory_limit" example:"128.0" description:"内存限制(MB)"`
}

// ServiceStatusResponse 服务状态响应
type ServiceStatusResponse struct {
	Service         Service               `json:"service" description:"服务基本信息"`
	TotalReplicas   int                   `json:"total_replicas" example:"3" description:"总副本数量"`
	HealthyReplicas int                   `json:"healthy_replicas" example:"3" description:"健康副本数量"`
	RunningReplicas int                   `json:"running_replicas" example:"2" description:"运行中副本数量"`
	StoppedReplicas int                   `json:"stopped_replicas" example:"1" description:"已停止副本数量"`
	FailedReplicas  int                   `json:"failed_replicas" example:"0" description:"失败副本数量"`
	Instances       []ServiceInstanceInfo `json:"instances" description:"实例详细信息列表"`
	LoadBalancer    string                `json:"load_balancer" example:"round_robin" description:"负载均衡策略"`
	AccessURL       string                `json:"access_url" example:"http://localhost:30000" description:"访问地址"`
	CreatedAt       time.Time             `json:"created_at" example:"2023-01-01T00:00:00Z" description:"创建时间"`
	UpdatedAt       time.Time             `json:"updated_at" example:"2023-01-01T00:00:00Z" description:"更新时间"`
}
