package dockerclient

import (
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/aichy126/igo"
	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/log"
	"github.com/davecgh/go-spew/spew"
)

var ctx context.IContext
var devContainers = &Service{
	Name:         "test-nginx",
	Image:        "nginx",
	Tag:          "alpine",
	PublicPort:   9200,
	InternalPort: 80,
}

func Init() {
	confPath := flag.String("config", "../../config.toml", "configure file")
	flag.Parse()

	igo.App = igo.NewApp(*confPath)
	ctx = context.Background()
}

// TestDockerClient 测试Docker客户端基础功能
func TestDockerClient(t *testing.T) {
	Init()

	// 创建Docker客户端
	client, err := NewDockerClient()
	if err != nil {
		t.Fatalf("创建Docker客户端失败: %v", err)
	}

	spew.Dump("===Docker客户端创建成功===", client.containerPrefix)
}

// TestGenerateContainerName 测试容器名称生成
func TestGenerateContainerName(t *testing.T) {
	Init()

	client, err := NewDockerClient()
	if err != nil {
		t.Fatalf("创建Docker客户端失败: %v", err)
	}

	// 测试容器名称生成
	containerName := client.generateContainerName("nginx-web", 8080, 30001, 0)

	spew.Dump("===容器名称生成测试===", containerName)
}

// TestParseContainerName 测试容器名称解析
func TestParseContainerName(t *testing.T) {
	Init()

	client, err := NewDockerClient()
	if err != nil {
		t.Fatalf("创建Docker客户端失败: %v", err)
	}

	// 测试容器名称解析
	containerName := "onedock-ai-shifu-cook-web-dev-p12013-c30004-0"
	info, err := client.ParseContainerName(containerName)
	if err != nil {
		t.Fatalf("容器名称解析失败: %v", err)
	}

	spew.Dump("===容器名称解析测试===", info)
}

func TestCreateContainer(t *testing.T) {
	Init()

	client, err := NewDockerClient()
	if err != nil {
		t.Fatalf("创建Docker客户端失败: %v", err)
	}

	dockerID, err := client.CreateContainer(ctx, devContainers, 0)
	if err != nil {
		log.Error("Docker", log.Any("Error", fmt.Sprintf("failed to create container: %v", err)))
	}
	fmt.Println("dockerID", dockerID)
	err = client.StartContainer(ctx, dockerID)
	if err != nil {
		log.Error("Docker", log.Any("Error", fmt.Sprintf("failed to start container: %v", err)))
	}
	// time.Sleep(time.Second * 3)
	// err = client.StopContainer(ctx, dockerID)
	// if err != nil {
	// 	log.Error("Docker", log.Any("Error", fmt.Sprintf("failed to stop container: %v", err)))
	// }
	// time.Sleep(time.Second * 1)
	// err = client.RemoveContainer(ctx, dockerID)
	// if err != nil {
	// 	log.Error("Docker", log.Any("Error", fmt.Sprintf("failed to remove container: %v", err)))
	// }
}

// ListContainers
func TestListContainers(t *testing.T) {
	Init()

	client, err := NewDockerClient()
	if err != nil {
		t.Fatalf("创建Docker客户端失败: %v", err)
	}
	list, err := client.ListContainers(ctx)
	if err != nil {
		t.Fatalf("获取容器列表失败: %v", err)
	}
	spew.Dump(list)
}

// InspectContainer
func TestInspectContainer(t *testing.T) {
	Init()

	client, err := NewDockerClient()
	if err != nil {
		t.Fatalf("创建Docker客户端失败: %v", err)
	}
	dockerID := "4fc8f3d869e56e73bbed170697c8ebc13027c51e7339635e64e926e9fd7cbeb8"
	info, err := client.InspectContainer(ctx, dockerID)
	if err != nil {
		t.Fatalf("获取容器列表失败: %v", err)
	}
	spew.Dump(info)
}

// GetNextReplicaIndex
func TestGetNextReplicaIndex(t *testing.T) {
	Init()

	client, err := NewDockerClient()
	if err != nil {
		t.Fatalf("创建Docker客户端失败: %v", err)
	}
	serviceName := devContainers.Name
	info, err := client.GetNextReplicaIndex(ctx, serviceName)
	if err != nil {
		t.Fatalf("获取容器列表失败: %v", err)
	}
	spew.Dump(info)
}

// TestSimpleScaleService 测试简化的扩缩容功能
func TestSimpleScaleService(t *testing.T) {
	Init()

	client, err := NewDockerClient()
	if err != nil {
		t.Fatalf("创建Docker客户端失败: %v", err)
	}

	serviceName := devContainers.Name

	// 测试扩容到3个副本
	targetReplicas := 3
	err = client.ScaleService(ctx, serviceName, targetReplicas)
	if err != nil {
		// 如果服务不存在，这是预期的错误
		if strings.Contains(err.Error(), "not found") {
			spew.Dump("===简化扩缩容测试===", "服务不存在（预期行为）:", err.Error())
		} else {
			t.Fatalf("扩缩容操作失败: %v", err)
		}
	} else {
		spew.Dump("===简化扩缩容测试===", "扩容操作完成，目标副本数:", targetReplicas)
	}
}

// TestExtractServiceFromContainer 测试从容器提取服务配置
func TestExtractServiceFromContainer(t *testing.T) {
	Init()

	client, err := NewDockerClient()
	if err != nil {
		t.Fatalf("创建Docker客户端失败: %v", err)
	}

	// 获取容器列表
	containers, err := client.ListContainers(ctx)
	if err != nil {
		t.Fatalf("获取容器列表失败: %v", err)
	}

	// 查找一个由我们管理的容器
	for _, container := range containers {
		service, err := client.ExtractServiceFromContainer(container)
		if err != nil {
			spew.Dump("===提取服务配置测试===", "提取失败:", err.Error())
			continue
		}

		spew.Dump("===提取服务配置测试===", "成功提取配置:", service)
		return
	}
	spew.Dump("===提取服务配置测试===", "没有找到可测试的容器")
}
