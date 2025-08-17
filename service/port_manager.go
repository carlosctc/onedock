package service

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/log"
	"github.com/aichy126/igo/util"
	"github.com/gin-gonic/gin"
)

// LoadBalanceStrategy 负载均衡策略类型
type LoadBalanceStrategy string

const (
	RoundRobin       LoadBalanceStrategy = "round_robin"
	LeastConnections LoadBalanceStrategy = "least_connections"
	Weighted         LoadBalanceStrategy = "weighted"
)

// Backend 后端服务器信息
type Backend struct {
	ContainerMapping *ContainerMapping
	Proxy            *httputil.ReverseProxy
	Active           bool
	Connections      int64
	Weight           int
	LastUsed         time.Time
}

// LoadBalancer 负载均衡器
type LoadBalancer struct {
	strategy LoadBalanceStrategy
	backends []*Backend
	current  int64
	mutex    sync.RWMutex
}

// PortManager 端口代理管理器
type PortManager struct {
	service   *Service
	servers   map[int]*http.Server           // publicPort -> HTTP服务器
	proxies   map[int]*httputil.ReverseProxy // 单副本代理
	balancers map[int]*LoadBalancer          // 多副本负载均衡器
	mutex     sync.RWMutex
}

// NewPortManager 创建端口管理器
func NewPortManager(service *Service) *PortManager {
	return &PortManager{
		service:   service,
		servers:   make(map[int]*http.Server),
		proxies:   make(map[int]*httputil.ReverseProxy),
		balancers: make(map[int]*LoadBalancer),
	}
}

// StartPortProxy 启动端口代理
func (pm *PortManager) StartPortProxy(ctx context.IContext, publicPort int) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// 检查是否已经存在
	if _, exists := pm.servers[publicPort]; exists {
		log.Info("PortManager", log.Any("Message", fmt.Sprintf("Port proxy already exists for port %d", publicPort)))
		return nil
	}

	// 获取容器映射
	mappings, err := pm.service.GetContainerMapping(ctx, publicPort)
	if err != nil {
		log.Error("PortManager", log.Any("Error", fmt.Sprintf("Failed to get container mapping for port %d: %v", publicPort, err)))
		return fmt.Errorf("failed to get container mapping: %w", err)
	}

	if len(mappings) == 0 {
		log.Warn("PortManager", log.Any("Message", fmt.Sprintf("No containers found for port %d", publicPort)))
		return fmt.Errorf("no containers found for port %d", publicPort)
	}

	// 根据容器数量决定代理类型
	if len(mappings) == 1 {
		// 单副本：使用直接代理
		return pm.startSingleProxy(publicPort, mappings[0])
	} else {
		// 多副本：使用负载均衡器
		return pm.startLoadBalancer(publicPort, mappings)
	}
}

// startSingleProxy 启动单副本代理
func (pm *PortManager) startSingleProxy(publicPort int, mapping *ContainerMapping) error {
	// 创建反向代理
	targetURL := fmt.Sprintf("http://localhost:%d", mapping.ContainerPort)
	target, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("failed to parse target URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("PortManager", log.Any("Error", fmt.Sprintf("Proxy error for port %d -> %d: %v", publicPort, mapping.ContainerPort, err)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(fmt.Sprintf("Service %s is unavailable", mapping.ServiceName)))
	}

	// 创建 HTTP 服务器
	router := gin.New()
	router.Use(gin.Recovery())
	router.NoRoute(gin.WrapH(proxy))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", publicPort),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// 存储代理和服务器
	pm.proxies[publicPort] = proxy
	pm.servers[publicPort] = server

	// 启动服务器
	go func() {
		log.Info("PortManager", log.Any("Message", fmt.Sprintf("Starting single proxy server: port %d -> container %d", publicPort, mapping.ContainerPort)))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("PortManager", log.Any("Error", fmt.Sprintf("Single proxy server error for port %d: %v", publicPort, err)))
		}
	}()

	log.Info("PortManager", log.Any("Message", fmt.Sprintf("Single proxy started for port %d -> container %d", publicPort, mapping.ContainerPort)))
	return nil
}

// startLoadBalancer 启动负载均衡器
func (pm *PortManager) startLoadBalancer(publicPort int, mappings []*ContainerMapping) error {
	// 获取负载均衡策略
	strategyConfig := util.ConfGetString("container.load_balance_strategy")
	strategy := LoadBalanceStrategy(strategyConfig)
	if strategy == "" {
		strategy = RoundRobin // 默认策略
	}

	// 创建负载均衡器
	balancer := &LoadBalancer{
		strategy: strategy,
		backends: make([]*Backend, 0, len(mappings)),
	}

	// 添加后端服务器
	for _, mapping := range mappings {
		backend, err := pm.createBackend(mapping)
		if err != nil {
			log.Error("PortManager", log.Any("Error", fmt.Sprintf("Failed to create backend for container %s: %v", mapping.ContainerID, err)))
			continue
		}
		balancer.backends = append(balancer.backends, backend)
	}

	if len(balancer.backends) == 0 {
		return fmt.Errorf("no valid backends for port %d", publicPort)
	}

	// 创建 HTTP 服务器
	router := gin.New()
	router.Use(gin.Recovery())
	router.NoRoute(func(c *gin.Context) {
		backend := balancer.SelectBackend(c.Request)
		if backend == nil {
			log.Error("PortManager", log.Any("Error", fmt.Sprintf("No available backend for port %d", publicPort)))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No available backends"})
			return
		}

		// 增加连接计数
		atomic.AddInt64(&backend.Connections, 1)
		defer atomic.AddInt64(&backend.Connections, -1)

		backend.LastUsed = time.Now()
		log.Debug("PortManager", log.Any("Message", fmt.Sprintf("Load balancing request: %s %s -> container %d", c.Request.Method, c.Request.URL.Path, backend.ContainerMapping.ContainerPort)))

		// 代理请求
		backend.Proxy.ServeHTTP(c.Writer, c.Request)
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", publicPort),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// 存储负载均衡器和服务器
	pm.balancers[publicPort] = balancer
	pm.servers[publicPort] = server

	// 启动服务器
	go func() {
		log.Info("PortManager", log.Any("Message", fmt.Sprintf("Starting load balancer: port %d with %d backends using %s strategy", publicPort, len(balancer.backends), strategy)))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("PortManager", log.Any("Error", fmt.Sprintf("Load balancer server error for port %d: %v", publicPort, err)))
		}
	}()

	log.Info("PortManager", log.Any("Message", fmt.Sprintf("Load balancer started for port %d with %d backends", publicPort, len(balancer.backends))))
	return nil
}

// createBackend 创建后端服务器
func (pm *PortManager) createBackend(mapping *ContainerMapping) (*Backend, error) {
	targetURL := fmt.Sprintf("http://localhost:%d", mapping.ContainerPort)
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("PortManager", log.Any("Error", fmt.Sprintf("Backend error for container %s: %v", mapping.ContainerID, err)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(fmt.Sprintf("Backend %s is unavailable", mapping.ContainerID)))
	}

	return &Backend{
		ContainerMapping: mapping,
		Proxy:            proxy,
		Active:           true,
		Weight:           100, // 默认权重
		LastUsed:         time.Now(),
	}, nil
}

// SelectBackend 选择后端服务器
func (lb *LoadBalancer) SelectBackend(r *http.Request) *Backend {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// 获取活跃后端
	activeBackends := make([]*Backend, 0)
	for _, backend := range lb.backends {
		if backend.Active {
			activeBackends = append(activeBackends, backend)
		}
	}

	if len(activeBackends) == 0 {
		return nil
	}

	switch lb.strategy {
	case RoundRobin:
		return lb.selectRoundRobin(activeBackends)
	case LeastConnections:
		return lb.selectLeastConnections(activeBackends)
	case Weighted:
		return lb.selectWeighted(activeBackends)
	default:
		return lb.selectRoundRobin(activeBackends)
	}
}

// selectRoundRobin 轮询选择
func (lb *LoadBalancer) selectRoundRobin(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}

	current := atomic.AddInt64(&lb.current, 1)
	return backends[(current-1)%int64(len(backends))]
}

// selectLeastConnections 最少连接选择
func (lb *LoadBalancer) selectLeastConnections(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}

	selected := backends[0]
	minConnections := atomic.LoadInt64(&selected.Connections)

	for _, backend := range backends[1:] {
		connections := atomic.LoadInt64(&backend.Connections)
		if connections < minConnections {
			minConnections = connections
			selected = backend
		}
	}

	return selected
}

// selectWeighted 权重选择
func (lb *LoadBalancer) selectWeighted(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}

	// 计算总权重
	totalWeight := 0
	for _, backend := range backends {
		totalWeight += backend.Weight
	}

	if totalWeight == 0 {
		return lb.selectRoundRobin(backends)
	}

	// 生成随机数选择
	target := rand.Intn(totalWeight)

	current := 0
	for _, backend := range backends {
		current += backend.Weight
		if current > target {
			return backend
		}
	}

	return backends[0]
}

// StopPortProxy 停止端口代理
func (pm *PortManager) StopPortProxy(publicPort int) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// 停止 HTTP 服务器
	if server, exists := pm.servers[publicPort]; exists {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Error("PortManager", log.Any("Error", fmt.Sprintf("Failed to shutdown server for port %d: %v", publicPort, err)))
		}
		delete(pm.servers, publicPort)
	}

	// 清理代理和负载均衡器
	delete(pm.proxies, publicPort)
	delete(pm.balancers, publicPort)

	log.Info("PortManager", log.Any("Message", fmt.Sprintf("Port proxy stopped for port %d", publicPort)))
	return nil
}

// UpdatePortProxy 更新端口代理
func (pm *PortManager) UpdatePortProxy(ctx context.IContext, publicPort int) error {
	// 先停止现有代理
	if err := pm.StopPortProxy(publicPort); err != nil {
		log.Error("PortManager", log.Any("Error", fmt.Sprintf("Failed to stop existing proxy for port %d: %v", publicPort, err)))
	}

	// 等待一小段时间确保端口释放
	time.Sleep(100 * time.Millisecond)

	// 重新启动代理
	return pm.StartPortProxy(ctx, publicPort)
}

// GetProxyStats 获取代理统计信息
func (pm *PortManager) GetProxyStats(ctx context.IContext) map[string]interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_proxies":  len(pm.servers),
		"single_proxies": len(pm.proxies),
		"load_balancers": len(pm.balancers),
		"proxy_details":  make([]map[string]interface{}, 0),
	}

	for port, server := range pm.servers {
		detail := map[string]interface{}{
			"public_port": port,
			"server_addr": server.Addr,
		}

		if _, isSingle := pm.proxies[port]; isSingle {
			detail["type"] = "single_proxy"
		} else if balancer, isBalancer := pm.balancers[port]; isBalancer {
			detail["type"] = "load_balancer"
			detail["strategy"] = balancer.strategy
			detail["backend_count"] = len(balancer.backends)

			backends := make([]map[string]interface{}, 0)
			for _, backend := range balancer.backends {
				backends = append(backends, map[string]interface{}{
					"container_id":   backend.ContainerMapping.ContainerID,
					"container_port": backend.ContainerMapping.ContainerPort,
					"active":         backend.Active,
					"connections":    atomic.LoadInt64(&backend.Connections),
					"weight":         backend.Weight,
					"last_used":      backend.LastUsed,
				})
			}
			detail["backends"] = backends
		}

		stats["proxy_details"] = append(stats["proxy_details"].([]map[string]interface{}), detail)
	}

	return stats
}

// Shutdown 关闭所有代理
func (pm *PortManager) Shutdown() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var errors []error
	for port, server := range pm.servers {
		if err := server.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown server for port %d: %w", port, err))
		}
	}

	// 清理所有映射
	pm.servers = make(map[int]*http.Server)
	pm.proxies = make(map[int]*httputil.ReverseProxy)
	pm.balancers = make(map[int]*LoadBalancer)

	log.Info("PortManager", log.Any("Message", "All port proxies shutdown"))

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}
	return nil
}
