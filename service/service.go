package service

import (
	"fmt"

	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/log"
	"github.com/aichy126/onedock/library/cache"
	"github.com/aichy126/onedock/library/dockerclient"
)

// Service
type Service struct {
	Cache        *cache.MemCache
	dockerClient *dockerclient.DockerClient
	PortManager  *PortProxyManager
}

// NewService
func NewService() *Service {
	docekrClient, err := dockerclient.NewDockerClient()
	if err != nil {
		log.Error("Docker", log.Any("Error", fmt.Sprintf("failed to create docker client: %v", err)))
		return nil
	}

	service := &Service{
		Cache:        cache.NewMemCache(),
		dockerClient: docekrClient,
	}

	// 初始化端口管理器
	service.PortManager = NewPortManager(service)

	// 恢复已存在的代理服务
	service.recoverPortProxies()

	return service
}

// recoverPortProxies 恢复所有已存在的端口代理服务
func (s *Service) recoverPortProxies() {
	ctx := context.Background()

	log.Info("PortProxy", log.Any("Message", "开始恢复端口代理服务..."))

	// 获取所有接管的服务
	services := s.ListServices(ctx)
	if len(services) == 0 {
		log.Info("PortProxy", log.Any("Message", "没有发现需要恢复的服务"))
		return
	}

	successCount := 0
	failureCount := 0

	// 遍历每个服务，重新启动端口代理
	for _, service := range services {
		if service.PublicPort <= 0 {
			continue
		}

		// 检查服务是否有运行的副本
		if service.Replicas <= 0 {
			continue
		}

		// 启动端口代理
		err := s.PortManager.StartPortProxy(ctx, service.PublicPort)
		if err != nil {
			failureCount++
		} else {
			successCount++
		}
	}
	log.Info("PortProxy",
		log.Any("Total", len(services)),
		log.Any("Success", successCount),
		log.Any("Failure", failureCount),
		log.Any("Message", "端口代理恢复完成"))
}
