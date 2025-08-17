package service

import (
	"strconv"

	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/util"
	"github.com/aichy126/onedock/models"
)

// ContainerMapping 容器映射信息
type ContainerMapping struct {
	PublicPort    int    `json:"public_port"`    // 对外暴露端口
	ContainerPort int    `json:"container_port"` // 容器映射端口
	ContainerID   string `json:"container_id"`   // 容器ID
	ServiceName   string `json:"service_name"`   // 服务名称
}

//PortMapping

// GetContainerMapping 获取端口的所有容器映射
// 缓存未命中时会从 Docker 实时查询并重新构建缓存
func (s *Service) GetContainerMapping(ctx context.IContext, publicPort int) ([]*ContainerMapping, error) {
	cacheKey := models.ContainerMappingKey + ":" + strconv.Itoa(publicPort)

	// 尝试从缓存获取
	var cachedList []*ContainerMapping
	err := s.Cache.Get(ctx, cacheKey, &cachedList)
	if err == nil && len(cachedList) > 0 {
		return cachedList, nil
	}

	// 缓存未命中，从 Docker 实时查询
	mappings, err := s.rebuildContainerMappingFromDocker(ctx, publicPort)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	if len(mappings) > 0 {
		cacheTime := util.ConfGetInt("container.cache_ttl")
		s.Cache.Set(ctx, cacheKey, mappings, cacheTime)
	}

	return mappings, nil
}

// DelContainerMapping 删除端口映射缓存
// 当容器被删除或服务停止时调用此方法清理缓存
func (s *Service) DelContainerMapping(ctx context.IContext, publicPort int) error {
	cacheKey := models.ContainerMappingKey + ":" + strconv.Itoa(publicPort)
	return s.Cache.Del(ctx, cacheKey)
}

// rebuildContainerMappingFromDocker 从 Docker 实时查询重建端口映射
// 这是缓存失效时的数据源，确保数据准确性
func (s *Service) rebuildContainerMappingFromDocker(ctx context.IContext, publicPort int) ([]*ContainerMapping, error) {

	// 获取所有容器
	allContainers, err := s.dockerClient.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	mappings := make([]*ContainerMapping, 0)

	// 遍历容器，查找匹配指定公共端口的容器
	for _, container := range allContainers {

		if container.State != "running" {
			continue
		}
		containerNameInfo, err := s.dockerClient.ParseContainerName(container.Name)
		if err != nil {
			continue
		}
		if containerNameInfo.PublicPort != publicPort {
			continue
		}

		mapping := &ContainerMapping{
			PublicPort:    publicPort,
			ContainerPort: containerNameInfo.ContainerPort,
			ContainerID:   container.ID,
			ServiceName:   containerNameInfo.ServiceName,
		}

		mappings = append(mappings, mapping)
	}

	return mappings, nil
}
