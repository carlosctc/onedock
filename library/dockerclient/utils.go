package dockerclient

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/aichy126/igo/log"
	"github.com/docker/docker/api/types/container"
)

// generateContainerName 生成标准格式的容器名称
// 格式: {prefix}-{serviceName}-p{publicPort}-c{containerPort}-{replicaIndex}
func (dc *DockerClient) generateContainerName(serviceName string, publicPort, containerPort, replicaIndex int) string {
	return fmt.Sprintf("%s-%s-p%d-c%d-%d", dc.containerPrefix, serviceName, publicPort, containerPort, replicaIndex)
}

// ParseContainerName 解析容器名称，提取服务信息
// 从标准格式的容器名称中解析出服务名、端口和副本信息
func (dc *DockerClient) ParseContainerName(containerName string) (*ContainerNameInfo, error) {
	if dc.containerPrefix == "" {
		return nil, fmt.Errorf("prefix cannot be empty")
	}

	// 检查是否以指定前缀开头
	if !strings.HasPrefix(containerName, dc.containerPrefix+"-") {
		return nil, fmt.Errorf("container name does not match prefix: %s", dc.containerPrefix)
	}

	// 移除前缀
	remaining := strings.TrimPrefix(containerName, dc.containerPrefix+"-")

	// 解析格式：serviceName-p{publicPort}-c{containerPort}-{replicaIndex}
	pattern := regexp.MustCompile(`^(.+)-p(\d+)-c(\d+)-(\d+)$`)
	matches := pattern.FindStringSubmatch(remaining)

	if len(matches) != 5 {
		return nil, fmt.Errorf("container name does not match expected format: %s", containerName)
	}

	serviceName := matches[1]
	publicPort, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid public port: %s", matches[2])
	}

	containerPort, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, fmt.Errorf("invalid container port: %s", matches[3])
	}

	replicaIndex, err := strconv.Atoi(matches[4])
	if err != nil {
		return nil, fmt.Errorf("invalid replica index: %s", matches[4])
	}

	return &ContainerNameInfo{
		ServiceName:   serviceName,
		PublicPort:    publicPort,
		ContainerPort: containerPort,
		ReplicaIndex:  replicaIndex,
	}, nil
}

// detectPlatform 检测运行平台并返回适合的容器配置
// 根据不同操作系统(macOS/Linux)返回对应的Docker主机配置
func (dc *DockerClient) detectPlatform() *container.HostConfig {
	baseConfig := &container.HostConfig{
		AutoRemove: false,
	}

	switch runtime.GOOS {
	case "darwin":
		log.Info("Docker", log.Any("Message", "检测到macOS环境，使用OrbStack/Docker Desktop兼容配置"))
		return baseConfig

	case "linux":
		log.Info("Docker", log.Any("Message", "检测到Linux环境，使用标准Docker配置"))
		baseConfig.RestartPolicy = container.RestartPolicy{
			Name: "unless-stopped",
		}
		return baseConfig

	default:
		log.Warn("Docker", log.Any("GOOS", runtime.GOOS), log.Any("Message", "未知运行环境，使用基础配置"))
		return baseConfig
	}
}

// ExtractServiceFromContainer 从容器中提取Service配置
// 根据容器的标签和配置信息重建Service结构体
func (dc *DockerClient) ExtractServiceFromContainer(container ContainerInfo) (*Service, error) {
	// 解析容器名称获取基本信息
	nameInfo, err := dc.ParseContainerName(container.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to parse container name: %w", err)
	}

	// 从标签中提取配置信息
	labels := container.Labels
	serviceName := labels[dc.containerPrefix+".service"]
	image := labels[dc.containerPrefix+".image"]
	tag := labels[dc.containerPrefix+".tag"]
	publicPortStr := labels[dc.containerPrefix+".public_port"]

	if serviceName == "" || image == "" || tag == "" {
		return nil, fmt.Errorf("container missing required labels")
	}

	publicPort, err := strconv.Atoi(publicPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid public port in labels: %s", publicPortStr)
	}

	// 从端口映射中提取内部端口
	internalPort := 80 // 默认值
	if len(container.Ports) > 0 {
		if port, err := strconv.Atoi(container.Ports[0].ContainerPort); err == nil {
			internalPort = port
		}
	}

	return &Service{
		Name:         serviceName,
		Image:        image,
		Tag:          tag,
		PublicPort:   publicPort,
		InternalPort: internalPort,
		DockerPort:   nameInfo.ContainerPort,
		Environment:  make(map[string]string), // 无法从容器中完整恢复，使用空值
		Volumes:      []VolumeMount{},         // 无法从容器中完整恢复，使用空值
		Command:      []string{},              // 无法从容器中完整恢复，使用空值
		WorkingDir:   "",                      // 无法从容器中完整恢复，使用空值
		Replicas:     1,                       // 单个容器的副本数为1
	}, nil
}

// findAvailablePortForService 查找服务的第一个可用端口号
// 从起始端口开始递增查找，跳过已被占用的端口
func (dc *DockerClient) findAvailablePortForService(containers []ContainerInfo, serviceName string) int {
	// 收集该服务已占用的所有端口
	usedPorts := make(map[int]bool)

	for _, container := range containers {
		containerInfo, err := dc.ParseContainerName(container.Name)
		if err != nil {
			continue
		}
		usedPorts[containerInfo.ContainerPort] = true
	}

	// 从起始端口开始查找第一个可用端口
	for port := dc.internalPortStart; ; port++ {
		if !usedPorts[port] && !dc.isPortOccupied(port) {
			return port
		}
	}
}

// isPortOccupied 检测指定端口是否被占用
// 通过尝试绑定端口来检测端口是否可用
func (dc *DockerClient) isPortOccupied(port int) bool {
	address := fmt.Sprintf(":%d", port)

	// 尝试监听TCP端口
	listener, err := net.Listen("tcp", address)
	if err != nil {
		// 如果监听失败，说明端口被占用
		return true
	}

	// 如果监听成功，立即关闭并返回端口可用
	defer listener.Close()
	return false
}

// readEnvFile 读取环境变量文件并返回键值对
// 简单实现：读取KEY=VALUE格式，跳过注释和空行
func (dc *DockerClient) readEnvFile(envFilePath string) (map[string]string, error) {
	if envFilePath == "" {
		return make(map[string]string), nil
	}

	file, err := os.Open(envFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open env file %s: %w", envFilePath, err)
	}
	defer file.Close()

	envVars := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析KEY=VALUE
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			// 移除引号
			if len(value) >= 2 {
				if (value[0] == '"' && value[len(value)-1] == '"') ||
					(value[0] == '\'' && value[len(value)-1] == '\'') {
					value = value[1 : len(value)-1]
				}
			}

			envVars[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read env file: %w", err)
	}

	return envVars, nil
}

// CompareServiceConfig 比较两个服务配置是否有差异
// 主要比较影响容器运行的关键参数：镜像、标签、环境变量、卷挂载、命令等
func (dc *DockerClient) CompareServiceConfig(oldService, newService *Service) bool {
	// 检查镜像和标签
	if oldService.Image != newService.Image || oldService.Tag != newService.Tag {
		return true
	}

	// 检查内部端口
	if oldService.InternalPort != newService.InternalPort {
		return true
	}

	// 检查环境变量
	if !dc.compareEnvironment(oldService.Environment, newService.Environment) {
		return true
	}

	// 检查卷挂载
	if !dc.compareVolumes(oldService.Volumes, newService.Volumes) {
		return true
	}

	// 检查启动命令
	if !dc.compareCommands(oldService.Command, newService.Command) {
		return true
	}

	// 检查工作目录
	if oldService.WorkingDir != newService.WorkingDir {
		return true
	}

	// 检查环境变量文件
	if oldService.EnvFile != newService.EnvFile {
		return true
	}

	return false // 没有差异
}

// compareEnvironment 比较环境变量映射
func (dc *DockerClient) compareEnvironment(old, new map[string]string) bool {
	if len(old) != len(new) {
		return false
	}

	for k, v := range old {
		if newV, exists := new[k]; !exists || newV != v {
			return false
		}
	}

	return true
}

// compareVolumes 比较卷挂载配置
func (dc *DockerClient) compareVolumes(old, new []VolumeMount) bool {
	if len(old) != len(new) {
		return false
	}

	// 创建映射以便于比较
	oldMap := make(map[string]VolumeMount)
	for _, vol := range old {
		oldMap[vol.Destination] = vol
	}

	for _, vol := range new {
		if oldVol, exists := oldMap[vol.Destination]; !exists ||
			oldVol.Source != vol.Source || oldVol.ReadOnly != vol.ReadOnly {
			return false
		}
	}

	return true
}

// compareCommands 比较启动命令
func (dc *DockerClient) compareCommands(old, new []string) bool {
	if len(old) != len(new) {
		return false
	}

	for i, cmd := range old {
		if cmd != new[i] {
			return false
		}
	}

	return true
}
