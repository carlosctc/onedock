package onedockclient

import (
	"time"
)

// ServiceStatus 服务状态
type ServiceStatus string

// VolumeMount 卷挂载配置
type VolumeMount struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Mode          string `json:"mode"` // "ro" 或 "rw"
}

// PortMapping 端口映射
type PortMapping struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port"`
	Protocol      string `json:"protocol"` // "tcp" 或 "udp"
}

// Service API 响应用的服务信息
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

type ServiceListResponse struct {
	Services []Service `json:"services"`
	Total    int       `json:"total"`
}

// ServiceRequest 服务部署/更新请求
type ServiceRequest struct {
	Name         string            `json:"name"`
	Image        string            `json:"image"`
	Tag          string            `json:"tag"`
	InternalPort int               `json:"internal_port"`
	Replicas     int               `json:"replicas,omitempty"`
	Environment  map[string]string `json:"environment,omitempty"`
	EnvFile      string            `json:"env_file,omitempty"`
	Volumes      []VolumeMount     `json:"volumes,omitempty"`
	Entrypoint   []string          `json:"entrypoint,omitempty"`
	Command      []string          `json:"command,omitempty"`
	WorkingDir   string            `json:"working_dir,omitempty"`
	PublicPort   int               `json:"public_port,omitempty"`
}

// ScaleRequest 扩缩容请求
type ScaleRequest struct {
	Replicas int `json:"replicas"`
}

// ServiceInstanceInfo 服务实例详细信息
type ServiceInstanceInfo struct {
	ID            string            `json:"id"`
	ContainerID   string            `json:"container_id"`
	ContainerName string            `json:"container_name"`
	ServiceName   string            `json:"service_name"`
	Status        string            `json:"status"`
	HealthStatus  string            `json:"health_status"`
	PublicPort    int               `json:"public_port"`
	ContainerPort int               `json:"container_port"`
	InternalPort  int               `json:"internal_port"`
	Image         string            `json:"image"`
	CreatedAt     time.Time         `json:"created_at"`
	StartedAt     time.Time         `json:"started_at"`
	IPAddress     string            `json:"ip_address"`
	Labels        map[string]string `json:"labels"`
	RestartCount  int               `json:"restart_count"`
	Uptime        string            `json:"uptime"`
	CPUUsage      float64           `json:"cpu_usage"`
	MemoryUsage   float64           `json:"memory_usage"`
	MemoryLimit   float64           `json:"memory_limit"`
}

// ServiceStatusResponse 服务状态响应
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

// ProxyStats 代理统计信息
type ProxyStats struct {
	TotalProxies      int                         `json:"total_proxies"`
	SingleProxies     int                         `json:"single_proxies"`
	LoadBalancers     int                         `json:"load_balancers"`
	ProxyDetails      []ProxyDetail               `json:"proxy_details"`
	LoadBalancerStats map[string]LoadBalancerStat `json:"load_balancer_stats"`
}

// ProxyDetail 代理详细信息
type ProxyDetail struct {
	PublicPort    int    `json:"public_port"`
	ServiceName   string `json:"service_name"`
	ProxyType     string `json:"proxy_type"`
	BackendCount  int    `json:"backend_count"`
	TotalRequests int64  `json:"total_requests"`
	ErrorCount    int64  `json:"error_count"`
	Status        string `json:"status"`
}

// LoadBalancerStat 负载均衡器统计
type LoadBalancerStat struct {
	Strategy      string                 `json:"strategy"`
	BackendCount  int                    `json:"backend_count"`
	TotalRequests int64                  `json:"total_requests"`
	BackendStats  map[string]BackendStat `json:"backend_stats"`
}

// BackendStat 后端统计
type BackendStat struct {
	Address     string `json:"address"`
	Requests    int64  `json:"requests"`
	Errors      int64  `json:"errors"`
	Connections int    `json:"connections"`
	Weight      int    `json:"weight"`
	Available   bool   `json:"available"`
}

// PingResponse Ping 响应
type PingResponse struct {
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version,omitempty"`
	Debug     map[string]interface{} `json:"debug,omitempty"`
}

type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}
