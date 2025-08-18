package service

import (
	"fmt"
	"time"

	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/log"
	"github.com/aichy126/onedock/library/dockerclient"
	"github.com/aichy126/onedock/models"
	"github.com/jinzhu/copier"
)

// UpdateService 更新服务 - 实现滚动更新逻辑
func (s *Service) UpdateService(ctx context.IContext, req *models.ServiceRequest) (*models.Service, error) {
	// 第一步：获取现有服务
	existingService := s.GetService(ctx, req.Name)
	if existingService == nil {
		return nil, fmt.Errorf("service %s not found", req.Name)
	}

	log.Info("Docker", log.Any("ServiceName", req.Name), log.Any("Message", "开始滚动更新服务"))

	// 第二步：构建新的服务配置
	newDockerService := &dockerclient.Service{}
	err := copier.Copy(newDockerService, req)
	if err != nil {
		return nil, fmt.Errorf("failed to copy service request: %w", err)
	}

	// 第三步：获取现有容器列表
	containers, err := s.dockerClient.ListContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// 第四步：找到此服务的所有容器并提取旧配置
	var serviceContainers []dockerclient.ContainerInfo
	var oldDockerService *dockerclient.Service

	for _, container := range containers {
		nameInfo, err := s.dockerClient.ParseContainerName(container.Name)
		if err != nil {
			continue
		}

		if nameInfo.ServiceName == req.Name {
			serviceContainers = append(serviceContainers, container)

			// 从第一个容器提取旧配置
			if oldDockerService == nil {
				oldDockerService, err = s.dockerClient.ExtractServiceFromContainer(container)
				if err != nil {
					log.Error("Docker", log.Any("Error", err), log.Any("Message", "提取服务配置失败"))
					continue
				}
			}
		}
	}

	if len(serviceContainers) == 0 {
		return nil, fmt.Errorf("no containers found for service %s", req.Name)
	}

	if oldDockerService == nil {
		return nil, fmt.Errorf("failed to extract old service configuration")
	}

	log.Info("Docker", log.Any("ServiceName", req.Name), log.Any("Message", "检测到配置变化，开始滚动更新"))

	// 第五步：逐个更新容器
	successCount := 0

	for _, container := range serviceContainers {
		nameInfo, err := s.dockerClient.ParseContainerName(container.Name)
		if err != nil {
			log.Error("Docker", log.Any("Error", err), log.Any("ContainerName", container.Name), log.Any("Message", "解析容器名称失败"))
			continue
		}

		// 使用UpdateContainer方法更新单个容器
		newContainerID, newPort, err := s.dockerClient.UpdateContainer(ctx, req.Name, newDockerService, nameInfo.ReplicaIndex)
		if err != nil {
			log.Error("Docker", log.Any("Error", err), log.Any("ReplicaIndex", nameInfo.ReplicaIndex), log.Any("Message", "容器更新失败"))
			continue
		}

		successCount++

		log.Info("Docker", log.Any("ServiceName", req.Name), log.Any("ReplicaIndex", nameInfo.ReplicaIndex),
			log.Any("NewContainer", newContainerID[:12]), log.Any("NewPort", newPort), log.Any("Message", "容器更新成功"))
	}

	if successCount == 0 {
		return nil, fmt.Errorf("all container updates failed for service %s", req.Name)
	}

	if successCount < len(serviceContainers) {
		log.Warn("Docker", log.Any("ServiceName", req.Name), log.Any("Total", len(serviceContainers)),
			log.Any("Success", successCount), log.Any("Message", "部分容器更新失败"))
	}

	// 第六步：更新端口代理

	//删除缓存
	s.DelContainerMapping(ctx, existingService.PublicPort)

	log.Info("Docker", log.Any("ServiceName", req.Name), log.Any("PublicPort", existingService.PublicPort), log.Any("Message", "更新端口代理"))
	if err := s.PortManager.UpdatePortProxy(ctx, existingService.PublicPort); err != nil {
		log.Error("Docker", log.Any("Error", err), log.Any("PublicPort", existingService.PublicPort), log.Any("Message", "更新端口代理失败"))
		// 端口代理更新失败不影响服务更新结果，记录日志即可
	}

	// 第七步：返回更新后的服务信息
	updatedService := &models.Service{
		ID:           existingService.ID,
		Name:         req.Name,
		Image:        req.Image,
		Tag:          req.Tag,
		Status:       models.StatusRunning,
		PublicPort:   existingService.PublicPort, // 保持公共端口不变
		InternalPort: req.InternalPort,
		Replicas:     existingService.Replicas, // 副本数保持不变
		CreatedAt:    existingService.CreatedAt,
		UpdatedAt:    time.Now(),
	}

	log.Info("Docker", log.Any("ServiceName", req.Name), log.Any("UpdatedContainers", successCount),
		log.Any("Message", "滚动更新完成"))

	return updatedService, nil
}
