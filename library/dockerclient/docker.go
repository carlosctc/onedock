package dockerclient

import (
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"

	"github.com/aichy126/igo/context"

	"github.com/aichy126/igo/log"
	"github.com/aichy126/onedock/utils"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// NewDockerClient 创建新的Docker客户端实例
// 参数:
//   - containerPrefix: 容器名称前缀，用于标识管理的容器
func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error("Docker", log.Any("Error", fmt.Sprintf("failed to create docker client: %v", err)))
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &DockerClient{
		cli:               cli,
		containerPrefix:   utils.ConfGetString("container.prefix"),
		internalPortStart: utils.ConfGetInt("container.internal_port_start"),
	}, nil
}

// PullImage 拉取Docker镜像
// 参数:
//   - ctx: 上下文对象，用于控制超时和取消操作
//   - imageName: 镜像名称
//   - tag: 镜像标签
func (dc *DockerClient) PullImage(ctx context.IContext, imageName, tag string) error {
	fullImage := fmt.Sprintf("%s:%s", imageName, tag)

	log.Info("Docker", log.Any("Image", fullImage), log.Any("Message", "开始拉取镜像"))

	reader, err := dc.cli.ImagePull(ctx, fullImage, image.PullOptions{})
	if err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("Image", fullImage), log.Any("Message", "镜像拉取失败"))
		return fmt.Errorf("failed to pull image %s: %w", fullImage, err)
	}
	defer reader.Close()

	// 读取拉取输出（可选，用于显示进度）
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("Message", "读取拉取输出失败"))
		return fmt.Errorf("failed to read pull output: %w", err)
	}

	log.Info("Docker", log.Any("Image", fullImage), log.Any("Message", "镜像拉取完成"))
	return nil
}

// CreateContainerWithReplica 创建带副本编号的容器
// 根据服务配置创建Docker容器，支持端口映射、环境变量、卷挂载等配置
// 参数:
//   - ctx: 上下文对象
//   - service: 服务配置信息
//   - replicaIndex: 副本编号，用于区分同一服务的不同实例
func (dc *DockerClient) CreateContainer(ctx context.IContext, service *Service, replicaIndex int) (string, error) {
	fullImage := fmt.Sprintf("%s:%s", service.Image, service.Tag)

	// 构建端口映射
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}

	// 容器内部端口
	containerPort := nat.Port(fmt.Sprintf("%d/tcp", service.InternalPort))
	exposedPorts[containerPort] = struct{}{}

	// 重新获取最新的容器列表以确保端口分配正确
	latestContainers, err := dc.ListContainers(ctx)
	if err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("ServiceName", service.Name), log.Any("Message", "获取容器列表失败"))
		return "", fmt.Errorf("获取容器列表失败")
	}

	// 自动分配新的端口（基于现有最大端口+1）
	canUsePort := dc.findAvailablePortForService(latestContainers, service.Name)
	service.DockerPort = canUsePort

	// Docker主机映射端口 - 绑定到0.0.0.0允许外部访问
	portBindings[containerPort] = []nat.PortBinding{
		{
			HostIP:   "0.0.0.0",
			HostPort: strconv.Itoa(service.DockerPort),
		},
	}

	// 处理环境变量：先读取EnvFile，再添加直接指定的Environment
	allEnvVars := make(map[string]string)

	// 1. 先从EnvFile读取环境变量
	if service.EnvFile != "" {
		envFileVars, err := dc.readEnvFile(service.EnvFile)
		if err != nil {
			log.Error("Docker", log.Any("Error", err), log.Any("EnvFile", service.EnvFile), log.Any("Message", "读取环境变量文件失败"))
			return "", fmt.Errorf("failed to read env file: %w", err)
		}
		for k, v := range envFileVars {
			allEnvVars[k] = v
		}
		log.Info("Docker", log.Any("EnvFile", service.EnvFile), log.Any("Count", len(envFileVars)), log.Any("Message", "成功读取环境变量文件"))
	}

	// 2. 直接指定的Environment会覆盖EnvFile中的同名变量
	for k, v := range service.Environment {
		allEnvVars[k] = v
	}

	// 3. 构建最终的环境变量列表
	env := make([]string, 0, len(allEnvVars))
	for k, v := range allEnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// 构建卷挂载
	binds := make([]string, 0, len(service.Volumes))
	for _, volume := range service.Volumes {
		bind := fmt.Sprintf("%s:%s", volume.Source, volume.Destination)
		if volume.ReadOnly {
			bind += ":ro"
		}
		binds = append(binds, bind)
	}

	// 构建标签
	labels := map[string]string{
		dc.containerPrefix + ".managed":     "true",
		dc.containerPrefix + ".service":     service.Name,
		dc.containerPrefix + ".image":       service.Image,
		dc.containerPrefix + ".tag":         service.Tag,
		dc.containerPrefix + ".public_port": strconv.Itoa(service.PublicPort),
		dc.containerPrefix + ".platform":    runtime.GOOS, // 记录运行平台
	}

	// 容器配置
	config := &container.Config{
		Image:        fullImage,
		Env:          env,
		ExposedPorts: exposedPorts,
		Labels:       labels,
		WorkingDir:   service.WorkingDir,
		Tty:          true, // -t: 分配一个伪TTY
		OpenStdin:    true, // -i: 保持STDIN开放
		AttachStdin:  true, // 附加到STDIN
		AttachStdout: true, // 附加到STDOUT
		AttachStderr: true, // 附加到STDERR
	}

	// 如果有自定义命令
	if len(service.Command) > 0 {
		config.Cmd = service.Command
	}

	// 如果有自定义入口点
	if len(service.Entrypoint) > 0 {
		config.Entrypoint = service.Entrypoint
	}

	// 获取平台适配的主机配置
	hostConfig := dc.detectPlatform()
	hostConfig.PortBindings = portBindings
	hostConfig.Binds = binds

	// 添加重启策略 --restart always
	hostConfig.RestartPolicy = container.RestartPolicy{
		Name: "always",
	}

	// 添加安全参数
	hostConfig.ReadonlyRootfs = false // 默认不启用只读文件系统，避免影响应用写入
	hostConfig.Privileged = false     // 禁用特权模式

	// 日志配置
	hostConfig.LogConfig = container.LogConfig{
		Type: "json-file",
		Config: map[string]string{
			"max-size": "10m", // 单个日志文件最大 10MB
			"max-file": "3",   // 保留 3 个日志文件
		},
	}

	// 拉取镜像
	if err := dc.PullImage(ctx, service.Image, service.Tag); err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("ReplicaIndex", replicaIndex), log.Any("Message", "拉取镜像失败"))
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// 创建容器 - 使用新的命名规则：prefix-serviceName-p{publicPort}-c{containerPort}-{replicaIndex}
	containerName := dc.generateContainerName(service.Name, service.PublicPort, service.DockerPort, replicaIndex)

	resp, err := dc.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("ContainerName", containerName), log.Any("Message", "容器创建失败"))
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	log.Info("Docker", log.Any("ContainerName", containerName), log.Any("ID", resp.ID[:12]), log.Any("Message", "容器创建成功"))
	return resp.ID, nil
}

// StartContainer 启动指定的Docker容器
// 参数:
//   - ctx: 上下文对象
//   - containerID: 容器ID
func (dc *DockerClient) StartContainer(ctx context.IContext, containerID string) error {
	log.Info("Docker", log.Any("ID", containerID[:12]), log.Any("Platform", runtime.GOOS), log.Any("Message", "启动容器"))

	err := dc.cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("ID", containerID[:12]), log.Any("Message", "容器启动失败"))
		return fmt.Errorf("failed to start container %s: %w", containerID[:12], err)
	}

	log.Info("Docker", log.Any("ID", containerID[:12]), log.Any("Message", "容器启动成功"))
	return nil
}

// StopContainer 停止指定的Docker容器
// 使用30秒超时进行优雅停止
// 参数:
//   - ctx: 上下文对象
//   - containerID: 容器ID
func (dc *DockerClient) StopContainer(ctx context.IContext, containerID string) error {
	timeout := 30 // 30秒超时
	err := dc.cli.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("ID", containerID[:12]), log.Any("Message", "容器停止失败"))
		return fmt.Errorf("failed to stop container %s: %w", containerID[:12], err)
	}

	log.Info("Docker", log.Any("ID", containerID[:12]), log.Any("Message", "容器停止成功"))
	return nil
}

// RemoveContainer 删除指定的Docker容器
// 强制删除，即使容器正在运行
// 参数:
//   - ctx: 上下文对象
//   - containerID: 容器ID
func (dc *DockerClient) RemoveContainer(ctx context.IContext, containerID string) error {
	err := dc.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: true, // 强制删除，即使容器正在运行
	})
	if err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("ID", containerID[:12]), log.Any("Message", "容器删除失败"))
		return fmt.Errorf("failed to remove container %s: %w", containerID[:12], err)
	}

	log.Info("Docker", log.Any("ID", containerID[:12]), log.Any("Message", "容器删除成功"))
	return nil
}

// ListContainers 列出所有管理的Docker容器
// 返回包含运行中和已停止的所有管理容器信息列表（自动过滤非管理容器）
// 参数:
//   - ctx: 上下文对象
func (dc *DockerClient) ListContainers(ctx context.IContext) ([]ContainerInfo, error) {
	containers, err := dc.cli.ContainerList(ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("Message", "获取容器列表失败"))
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, cont := range containers {
		// 获取容器名称（去掉前导斜杠）
		name := ""
		if len(cont.Names) > 0 {
			name = strings.TrimPrefix(cont.Names[0], "/")
		}

		// 只处理管理的容器
		_, err := dc.ParseContainerName(name)
		if err != nil {
			continue // 跳过非管理的容器
		}

		// 解析端口映射
		ports := make([]PortMapping, 0, len(cont.Ports))
		for _, port := range cont.Ports {
			if port.PublicPort > 0 {
				ports = append(ports, PortMapping{
					HostPort:      strconv.Itoa(int(port.PublicPort)),
					ContainerPort: strconv.Itoa(int(port.PrivatePort)),
					Protocol:      port.Type,
				})
			}
		}

		info := ContainerInfo{
			ID:        cont.ID,
			Name:      name,
			Image:     cont.Image,
			Status:    cont.Status,
			State:     cont.State,
			Ports:     ports,
			Labels:    cont.Labels,
			CreatedAt: fmt.Sprintf("%d", cont.Created),
		}

		result = append(result, info)
	}

	return result, nil
}

// InspectContainer 检查指定容器的详细信息
// 返回容器的完整配置和状态信息
// 参数:
//   - ctx: 上下文对象
//   - containerID: 容器ID
func (dc *DockerClient) InspectContainer(ctx context.IContext, containerID string) (*ContainerInfo, error) {
	inspect, err := dc.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("ID", containerID[:12]), log.Any("Message", "检查容器详情失败"))
		return nil, fmt.Errorf("failed to inspect container %s: %w", containerID[:12], err)
	}

	// 解析端口映射
	ports := make([]PortMapping, 0)
	if inspect.NetworkSettings != nil && inspect.NetworkSettings.Ports != nil {
		for containerPort, bindings := range inspect.NetworkSettings.Ports {
			if len(bindings) > 0 {
				for _, binding := range bindings {
					ports = append(ports, PortMapping{
						HostPort:      binding.HostPort,
						ContainerPort: containerPort.Port(),
						Protocol:      containerPort.Proto(),
					})
				}
			}
		}
	}

	// 获取容器名称
	name := strings.TrimPrefix(inspect.Name, "/")

	info := &ContainerInfo{
		ID:        inspect.ID,
		Name:      name,
		Image:     inspect.Config.Image,
		Status:    inspect.State.Status,
		State:     inspect.State.Status,
		Ports:     ports,
		Labels:    inspect.Config.Labels,
		CreatedAt: inspect.Created,
	}

	return info, nil
}

// GetNextReplicaIndex 获取服务的下一个可用副本编号
// 通过扫描现有容器，找到指定服务的第一个未使用的副本编号
// 参数:
//   - ctx: 上下文对象
//   - serviceName: 服务名称
func (dc *DockerClient) GetNextReplicaIndex(ctx context.IContext, serviceName string) (int, error) {
	containers, err := dc.ListContainers(ctx)
	if err != nil {
		return 0, err
	}

	usedIndexes := make(map[int]bool)

	for _, container := range containers {
		// 使用解析函数
		containerInfo, err := dc.ParseContainerName(container.Name)
		if err != nil {
			continue // 跳过无法解析的容器名
		}

		// 只处理同一服务的容器
		if containerInfo.ServiceName == serviceName {
			usedIndexes[containerInfo.ReplicaIndex] = true
		}
	}

	// 找到第一个未使用的编号
	for i := 0; ; i++ {
		if !usedIndexes[i] {
			return i, nil
		}
	}
}

// ScaleService 缩放服务副本数量
// 简化的扩缩容接口，只需要服务名和目标副本数
// 参数:
//   - ctx: 上下文对象
//   - serviceName: 服务名称
//   - targetReplicas: 目标副本数量
func (dc *DockerClient) ScaleService(ctx context.IContext, serviceName string, targetReplicas int) error {
	// 第一步：查看当前服务容器数量
	containers, err := dc.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// 找到服务的容器
	var serviceContainers []ContainerInfo
	for _, container := range containers {
		containerInfo, err := dc.ParseContainerName(container.Name)
		if err != nil {
			continue // 跳过无法解析的容器名
		}

		if containerInfo.ServiceName == serviceName {
			serviceContainers = append(serviceContainers, container)
		}
	}

	// 检查服务是否存在
	if len(serviceContainers) == 0 {
		return fmt.Errorf("service %s not found, no containers exist", serviceName)
	}

	currentReplicas := len(serviceContainers)

	// 第二步：从其中一个容器提取Service配置
	serviceConfig, err := dc.ExtractServiceFromContainer(serviceContainers[0])
	if err != nil {
		return fmt.Errorf("failed to extract service config from container: %w", err)
	}

	// 第三步：根据当前副本数与目标副本数执行扩容或缩容
	if targetReplicas > currentReplicas {
		// 扩容
		return dc.scaleUp(ctx, serviceConfig, currentReplicas, targetReplicas)
	} else {
		// 缩容
		return dc.scaleDown(ctx, serviceName, serviceContainers, targetReplicas)
	}
}

// scaleUp 扩容操作 - 创建新的副本容器
// 参数:
//   - ctx: 上下文对象
//   - serviceConfig: 服务配置
//   - currentReplicas: 当前副本数
//   - targetReplicas: 目标副本数
func (dc *DockerClient) scaleUp(ctx context.IContext, serviceConfig *Service, currentReplicas, targetReplicas int) error {
	for i := currentReplicas; i < targetReplicas; i++ {
		// 获取下一个可用的副本编号
		replicaIndex, err := dc.GetNextReplicaIndex(ctx, serviceConfig.Name)
		if err != nil {
			log.Error("Docker", log.Any("Error", err), log.Any("ServiceName", serviceConfig.Name), log.Any("Message", "获取副本索引失败"))
			continue
		}

		// 重新获取最新的容器列表以确保端口分配正确
		latestContainers, err := dc.ListContainers(ctx)
		if err != nil {
			log.Error("Docker", log.Any("Error", err), log.Any("ServiceName", serviceConfig.Name), log.Any("Message", "获取容器列表失败"))
			continue
		}

		// 自动分配新的端口（基于现有最大端口+1）
		canUsePort := dc.findAvailablePortForService(latestContainers, serviceConfig.Name)
		newDockerPort := canUsePort

		// 创建副本服务配置
		replicaService := &Service{
			Name:         serviceConfig.Name,
			Image:        serviceConfig.Image,
			Tag:          serviceConfig.Tag,
			PublicPort:   serviceConfig.PublicPort,
			InternalPort: serviceConfig.InternalPort,
			DockerPort:   newDockerPort,
			Environment:  serviceConfig.Environment,
			Volumes:      serviceConfig.Volumes,
			Entrypoint:   serviceConfig.Entrypoint,
			Command:      serviceConfig.Command,
			WorkingDir:   serviceConfig.WorkingDir,
			Replicas:     1,
		}

		// 创建容器
		containerID, err := dc.CreateContainer(ctx, replicaService, replicaIndex)
		if err != nil {
			log.Error("Docker", log.Any("Error", err), log.Any("ReplicaIndex", replicaIndex), log.Any("Message", "创建容器失败"))
			continue
		}

		// 启动容器
		if err := dc.StartContainer(ctx, containerID); err != nil {
			log.Error("Docker", log.Any("Error", err), log.Any("ContainerID", containerID[:12]), log.Any("Message", "启动容器失败"))
			// 清理失败的容器
			dc.RemoveContainer(ctx, containerID)
			continue
		}
	}

	return nil
}

// scaleDown 缩容操作 - 删除多余的副本容器
// 参数:
//   - ctx: 上下文对象
//   - serviceName: 服务名称
//   - serviceContainers: 服务的所有容器
//   - targetReplicas: 目标副本数
func (dc *DockerClient) scaleDown(ctx context.IContext, serviceName string, serviceContainers []ContainerInfo, targetReplicas int) error {
	currentReplicas := len(serviceContainers)
	containersToRemove := currentReplicas - targetReplicas
	removed := 0

	// 优先删除索引较高的容器（保留索引较低的）
	for i := len(serviceContainers) - 1; i >= 0 && removed < containersToRemove; i-- {
		container := serviceContainers[i]

		if err := dc.removeReplica(ctx, container); err != nil {
			log.Error("Docker", log.Any("Error", err), log.Any("ContainerName", container.Name), log.Any("Message", "删除副本失败"))
		} else {
			removed++
			log.Info("Docker", log.Any("ServiceName", serviceName), log.Any("ContainerName", container.Name),
				log.Any("Message", "成功删除副本"))
		}
	}

	if removed < containersToRemove {
		log.Warn("Docker", log.Any("ServiceName", serviceName), log.Any("Expected", containersToRemove),
			log.Any("Actual", removed), log.Any("Message", "部分容器删除失败"))
	}

	return nil
}

// removeReplica 删除单个副本容器
// 参数:
//   - ctx: 上下文对象
//   - container: 要删除的容器信息
func (dc *DockerClient) removeReplica(ctx context.IContext, container ContainerInfo) error {
	// 停止容器
	if err := dc.StopContainer(ctx, container.ID); err != nil {
		log.Warn("Docker", log.Any("Error", err), log.Any("ContainerID", container.ID[:12]), log.Any("Message", "停止容器失败"))
	}

	// 删除容器
	if err := dc.RemoveContainer(ctx, container.ID); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}

// UpdateContainer 滚动更新容器 - 创建新容器替换旧容器
// 此方法实现零停机的滚动更新：创建新容器，启动成功后删除旧容器
// 参数:
//   - ctx: 上下文对象
//   - serviceName: 服务名称
//   - newService: 新的服务配置
//   - replicaIndex: 要更新的副本索引
func (dc *DockerClient) UpdateContainer(ctx context.IContext, serviceName string, newService *Service, replicaIndex int) (string, int, error) {
	// 第一步：查找要更新的旧容器
	containers, err := dc.ListContainers(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("failed to list containers: %w", err)
	}

	var oldContainer *ContainerInfo
	for _, container := range containers {
		containerInfo, err := dc.ParseContainerName(container.Name)
		if err != nil {
			continue
		}

		if containerInfo.ServiceName == serviceName && containerInfo.ReplicaIndex == replicaIndex {
			oldContainer = &container
			break
		}
	}

	if oldContainer == nil {
		return "", 0, fmt.Errorf("container for service %s replica %d not found", serviceName, replicaIndex)
	}

	log.Info("Docker", log.Any("ServiceName", serviceName), log.Any("ReplicaIndex", replicaIndex),
		log.Any("OldContainer", oldContainer.ID[:12]), log.Any("Message", "开始滚动更新容器"))

	// 第二步：为新容器分配端口
	latestContainers, err := dc.ListContainers(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get latest containers: %w", err)
	}

	newDockerPort := dc.findAvailablePortForService(latestContainers, serviceName)

	// 第三步：创建新服务配置（使用新端口）
	updateService := &Service{
		Name:         newService.Name,
		Image:        newService.Image,
		Tag:          newService.Tag,
		PublicPort:   newService.PublicPort,
		InternalPort: newService.InternalPort,
		DockerPort:   newDockerPort,
		Environment:  newService.Environment,
		EnvFile:      newService.EnvFile,
		Volumes:      newService.Volumes,
		Entrypoint:   newService.Entrypoint,
		Command:      newService.Command,
		WorkingDir:   newService.WorkingDir,
		Replicas:     1,
	}

	// 第四步：拉取新镜像
	log.Info("Docker", log.Any("Image", fmt.Sprintf("%s:%s", updateService.Image, updateService.Tag)),
		log.Any("Message", "开始拉取新镜像"))
	if err := dc.PullImage(ctx, updateService.Image, updateService.Tag); err != nil {
		return "", 0, fmt.Errorf("failed to pull new image: %w", err)
	}

	// 第五步：创建新容器
	newContainerID, err := dc.CreateContainer(ctx, updateService, replicaIndex)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create new container: %w", err)
	}

	// 第六步：启动新容器
	if err := dc.StartContainer(ctx, newContainerID); err != nil {
		// 清理失败的新容器
		dc.RemoveContainer(ctx, newContainerID)
		return "", 0, fmt.Errorf("failed to start new container: %w", err)
	}

	log.Info("Docker", log.Any("ServiceName", serviceName), log.Any("ReplicaIndex", replicaIndex),
		log.Any("NewContainer", newContainerID[:12]), log.Any("NewPort", newDockerPort),
		log.Any("Message", "新容器启动成功"))

	// 第七步：等待一段时间确保新容器稳定运行
	// TODO: 这里可以添加健康检查逻辑
	// time.Sleep(5 * time.Second)

	// 第八步：停止旧容器
	log.Info("Docker", log.Any("OldContainer", oldContainer.ID[:12]), log.Any("Message", "停止旧容器"))
	if err := dc.StopContainer(ctx, oldContainer.ID); err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("OldContainer", oldContainer.ID[:12]),
			log.Any("Message", "停止旧容器失败，但新容器已启动"))
	}

	// 第九步：删除旧容器
	if err := dc.RemoveContainer(ctx, oldContainer.ID); err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("OldContainer", oldContainer.ID[:12]),
			log.Any("Message", "删除旧容器失败，但新容器已启动"))
	} else {
		log.Info("Docker", log.Any("OldContainer", oldContainer.ID[:12]), log.Any("Message", "旧容器已删除"))
	}

	log.Info("Docker", log.Any("ServiceName", serviceName), log.Any("ReplicaIndex", replicaIndex),
		log.Any("NewContainer", newContainerID[:12]), log.Any("Message", "容器滚动更新完成"))

	return newContainerID, newDockerPort, nil
}
