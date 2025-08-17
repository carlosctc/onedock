package service

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/log"
	"github.com/aichy126/onedock/library/dockerclient"
	"github.com/aichy126/onedock/models"
	"github.com/jinzhu/copier"
)

// DeployOrUpdateService 部署或更新服务
func (s *Service) DeployOrUpdateService(ctx context.IContext, req *models.ServiceRequest) (*models.Service, error) {
	// 检查服务是否存在
	existingService := s.GetService(ctx, req.Name)
	if existingService != nil {
		// 服务已存在，执行更新逻辑
		log.Info("Docker", log.Any("ServiceName", req.Name), log.Any("Message", "服务已存在，开始执行滚动更新"))
		return s.UpdateService(ctx, req)
	}

	// 设置默认值
	if req.PublicPort == 0 {
		return nil, fmt.Errorf("public port cannot be empty")
	}

	if req.Replicas == 0 {
		req.Replicas = 1
	}

	// 构建dockerclient.Service（端口由dockerclient内部分配）
	dockerService := &dockerclient.Service{}
	err := copier.Copy(dockerService, req)
	if err != nil {
		return nil, fmt.Errorf("failed to copy service request: %w", err)
	}

	// 创建容器（镜像拉取在 CreateContainer 中统一处理）
	containerID, err := s.dockerClient.CreateContainer(ctx, dockerService, 0)
	if err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("Message", "创建容器失败"))
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// 启动容器
	err = s.dockerClient.StartContainer(ctx, containerID)
	if err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("ContainerID", containerID[:12]), log.Any("Message", "启动容器失败"))
		// 清理失败的容器
		s.dockerClient.RemoveContainer(ctx, containerID)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// 如果需要多个副本，使用dockerclient的扩缩容功能
	if dockerService.Replicas > 1 {
		err = s.dockerClient.ScaleService(ctx, dockerService.Name, dockerService.Replicas)
		if err != nil {
			log.Error("Docker", log.Any("Error", err), log.Any("TargetReplicas", dockerService.Replicas), log.Any("Message", "扩展副本失败"))
			// 如果扩容失败，保持单个容器运行
		}
	}

	// 返回服务信息
	service := &models.Service{
		ID:           fmt.Sprintf("svc_%d", time.Now().Unix()),
		Name:         dockerService.Name,
		Image:        dockerService.Image,
		Tag:          dockerService.Tag,
		Status:       models.StatusRunning,
		PublicPort:   dockerService.PublicPort,
		InternalPort: dockerService.InternalPort,
		Replicas:     dockerService.Replicas,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 启动端口代理
	if err := s.PortManager.StartPortProxy(ctx, dockerService.PublicPort); err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("PublicPort", dockerService.PublicPort), log.Any("Message", "启动端口代理失败"))
		// 端口代理失败不影响服务部署，记录日志即可
	} else {
		log.Info("Docker", log.Any("PublicPort", dockerService.PublicPort), log.Any("ServiceName", dockerService.Name), log.Any("Message", "端口代理启动成功"))
	}

	return service, nil
}

// ListServices 列出所有服务
func (s *Service) ListServices(ctx context.IContext) []*models.Service {
	// 直接从dockerclient获取管理的容器列表（已过滤）
	containers, err := s.dockerClient.ListContainers(ctx)
	if err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("Message", "获取容器列表失败"))
		return []*models.Service{}
	}

	// 使用公共方法处理容器到服务的转换
	serviceMap := s.processContainersToServices(containers)

	// 转换为切片
	services := make([]*models.Service, 0, len(serviceMap))
	for _, service := range serviceMap {
		services = append(services, service)
	}

	return services
}

// GetService 获取服务详情
func (s *Service) GetService(ctx context.IContext, name string) *models.Service {
	services := s.ListServices(ctx)

	for _, service := range services {
		if service.Name == name {
			return service
		}
	}

	return nil
}

// DeleteService 删除服务
func (s *Service) DeleteService(ctx context.IContext, name string) error {
	// 直接调用扩缩容功能，设置为0副本即删除所有容器
	// 删除代理的逻辑统一在 ScaleService 中处理
	return s.ScaleService(ctx, name, 0)
}

// GetServiceStatus 获取服务状态
func (s *Service) GetServiceStatus(ctx context.IContext, name string) (*models.ServiceStatusResponse, error) {
	// 直接从dockerclient获取管理的容器列表（已过滤）
	containers, err := s.dockerClient.ListContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// 使用公共方法处理容器到服务的转换
	serviceMap := s.processContainersToServices(containers)

	// 获取指定的服务
	service, exists := serviceMap[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	var instances []models.ServiceInstanceInfo
	runningCount := 0
	stoppedCount := 0
	healthyCount := 0

	// 遍历容器，找到指定服务的实例
	for _, container := range containers {
		// 使用dockerclient的解析方法
		nameInfo, err := s.dockerClient.ParseContainerName(container.Name)
		if err != nil {
			continue // 跳过无法解析的容器
		}

		if nameInfo.ServiceName == name {

			// 解析端口信息
			containerPort := 0
			if len(container.Ports) > 0 {
				if port, err := strconv.Atoi(container.Ports[0].HostPort); err == nil {
					containerPort = port
				}
			}

			// 创建实例信息
			instance := models.ServiceInstanceInfo{
				ID:            container.ID[:12],
				ContainerID:   container.ID,
				ContainerName: container.Name,
				ServiceName:   name,
				Status:        container.Status,
				HealthStatus:  "unknown", // 暂时设为unknown
				PublicPort:    service.PublicPort,
				ContainerPort: containerPort,
				InternalPort:  service.InternalPort,
				Image:         container.Image,
				Labels:        container.Labels,
				RestartCount:  0, // 暂时设为0
				Uptime:        "",
				CPUUsage:      0.0,
				MemoryUsage:   0.0,
				MemoryLimit:   0.0,
			}

			if container.CreatedAt != "" {
				if createdTime, err := time.Parse(time.RFC3339, container.CreatedAt); err == nil {
					instance.CreatedAt = createdTime
					instance.StartedAt = createdTime
				}
			}

			instances = append(instances, instance)

			// 统计状态
			if container.State == "running" {
				runningCount++
				healthyCount++ // 简单地认为运行中的容器是健康的
				service.Status = models.StatusRunning
			} else {
				stoppedCount++
			}
		}
	}

	// 构建响应
	status := &models.ServiceStatusResponse{
		Service:         *service,
		TotalReplicas:   len(instances),
		HealthyReplicas: healthyCount,
		RunningReplicas: runningCount,
		StoppedReplicas: stoppedCount,
		FailedReplicas:  0, // 暂时设为0
		Instances:       instances,
		LoadBalancer:    "round_robin", // 默认负载均衡策略
		AccessURL:       fmt.Sprintf("http://localhost:%d", service.PublicPort),
		CreatedAt:       service.CreatedAt,
		UpdatedAt:       service.UpdatedAt,
	}

	return status, nil
}

// ScaleService 服务扩缩容 - 直接调用dockerclient
func (s *Service) ScaleService(ctx context.IContext, name string, replicas int) error {
	// 获取服务信息以确定公共端口
	service := s.GetService(ctx, name)
	if service == nil {
		return fmt.Errorf("service %s not found", name)
	}

	// 执行扩缩容操作
	err := s.dockerClient.ScaleService(ctx, name, replicas)
	if err != nil {
		return err
	}
	s.DelContainerMapping(ctx, service.PublicPort)

	if replicas == 0 {
		// 副本数为 0，删除服务，停止端口代理
		if err := s.PortManager.StopPortProxy(service.PublicPort); err != nil {
			log.Error("Docker", log.Any("Error", err), log.Any("PublicPort", service.PublicPort), log.Any("ServiceName", name), log.Any("Message", "停止端口代理失败"))
			// 端口代理停止失败不影响服务删除，记录日志即可
		} else {
			log.Info("Docker", log.Any("PublicPort", service.PublicPort), log.Any("ServiceName", name), log.Any("Message", "端口代理停止成功"))
		}

		// 清理端口映射缓存
		if err := s.DelContainerMapping(ctx, service.PublicPort); err != nil {
			log.Error("Docker", log.Any("Error", err), log.Any("PublicPort", service.PublicPort), log.Any("Message", "清理端口映射缓存失败"))
		}
	} else {
		// 副本数大于 0，更新端口代理以适应新的副本数
		if err := s.PortManager.UpdatePortProxy(ctx, service.PublicPort); err != nil {
			log.Error("Docker", log.Any("Error", err), log.Any("PublicPort", service.PublicPort), log.Any("ServiceName", name), log.Any("Replicas", replicas), log.Any("Message", "更新端口代理失败"))
			// 端口代理更新失败不影响扩缩容，记录日志即可
		}

		// 清理端口映射缓存，强制重新查询
		if err := s.DelContainerMapping(ctx, service.PublicPort); err != nil {
			log.Error("Docker", log.Any("Error", err), log.Any("PublicPort", service.PublicPort), log.Any("Message", "清理端口映射缓存失败"))
		}
	}

	return nil
}

// 辅助方法

// processContainersToServices 处理容器列表，按服务分组并返回服务映射
func (s *Service) processContainersToServices(containers []dockerclient.ContainerInfo) map[string]*models.Service {
	serviceMap := make(map[string]*models.Service)

	for _, container := range containers {
		// 使用dockerclient的解析方法提取服务名称
		nameInfo, err := s.dockerClient.ParseContainerName(container.Name)
		if err != nil {
			continue // 跳过无法解析的容器
		}

		// 获取或创建服务对象
		service, exists := serviceMap[nameInfo.ServiceName]
		if !exists {
			service = s.createServiceFromContainer(container)
			serviceMap[nameInfo.ServiceName] = service
		} else {
			// 更新副本数
			service.Replicas++
		}

		// 更新服务状态
		if container.State == "running" && service.Status != models.StatusRunning {
			service.Status = models.StatusRunning
		}
	}

	return serviceMap
}

// createServiceFromContainer 从容器信息创建服务对象
func (s *Service) createServiceFromContainer(container dockerclient.ContainerInfo) *models.Service {
	// 首先尝试使用 dockerclient 的方法提取服务信息
	dockerService, err := s.dockerClient.ExtractServiceFromContainer(container)
	if err != nil {
		// 如果无法从标签中提取，则使用兼容逻辑
		return s.createServiceFromContainerFallback(container)
	}

	// 转换为 models.Service
	service := &models.Service{
		ID:           container.ID[:12],
		Name:         dockerService.Name,
		Image:        dockerService.Image,
		Tag:          dockerService.Tag,
		Status:       models.ServiceStatus(container.State),
		PublicPort:   dockerService.PublicPort,
		InternalPort: dockerService.InternalPort,
		Replicas:     1, // 初始设为1，后续会更新
	}

	if container.CreatedAt != "" {
		if createdTime, err := time.Parse(time.RFC3339, container.CreatedAt); err == nil {
			service.CreatedAt = createdTime
			service.UpdatedAt = createdTime
		}
	}

	return service
}

// createServiceFromContainerFallback 兼容旧容器的回退方法
func (s *Service) createServiceFromContainerFallback(container dockerclient.ContainerInfo) *models.Service {
	// 解析端口信息
	internalPort := 0
	publicPort := 0

	if len(container.Ports) > 0 {
		if port, err := strconv.Atoi(container.Ports[0].ContainerPort); err == nil {
			internalPort = port
		}
	}

	// 从标签中获取公共端口
	if container.Labels != nil {
		if portStr, exists := container.Labels["public_port"]; exists {
			if port, err := strconv.Atoi(portStr); err == nil {
				publicPort = port
			}
		}
	}

	// 解析镜像名和标签
	image := container.Image
	tag := "latest"
	if colonIndex := len(image) - 1; colonIndex > 0 {
		for i := colonIndex; i >= 0; i-- {
			if image[i] == ':' {
				tag = image[i+1:]
				image = image[:i]
				break
			}
		}
	}

	// 解析容器名称获取服务名
	nameInfo, err := s.dockerClient.ParseContainerName(container.Name)
	serviceName := ""
	if err == nil {
		serviceName = nameInfo.ServiceName
	}

	service := &models.Service{
		ID:           container.ID[:12],
		Name:         serviceName,
		Image:        image,
		Tag:          tag,
		Status:       models.ServiceStatus(container.State),
		PublicPort:   publicPort,
		InternalPort: internalPort,
		Replicas:     1, // 初始设为1，后续会更新
	}

	if container.CreatedAt != "" {
		if createdTime, err := time.Parse(time.RFC3339, container.CreatedAt); err == nil {
			service.CreatedAt = createdTime
			service.UpdatedAt = createdTime
		}
	}

	return service
}
