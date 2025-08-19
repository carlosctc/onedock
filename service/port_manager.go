package service

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	igoContext "github.com/aichy126/igo/context"
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

// PortProxy 单个端口的代理实例
type PortProxy struct {
	publicPort int
	server     *http.Server
	proxyType  string // "single" 或 "load_balancer"
	cancel     context.CancelFunc
	ctx        context.Context
	
	// 具体代理实现（二选一）
	singleProxy *httputil.ReverseProxy
	balancer    *LoadBalancer
}

// PortProxyManager 端口代理管理器（轻量化）
type PortProxyManager struct {
	service *Service
	proxies map[int]*PortProxy // publicPort -> 独立的端口代理
	mutex   sync.RWMutex
}

// NewPortManager 创建端口代理管理器
func NewPortManager(service *Service) *PortProxyManager {
	return &PortProxyManager{
		service: service,
		proxies: make(map[int]*PortProxy),
	}
}

// StartPortProxy 启动端口代理
func (ppm *PortProxyManager) StartPortProxy(ctx igoContext.IContext, publicPort int) error {
	ppm.mutex.Lock()
	defer ppm.mutex.Unlock()

	// 检查是否已经存在
	if _, exists := ppm.proxies[publicPort]; exists {
		log.Info("PortProxyManager", log.Any("Message", fmt.Sprintf("Port proxy already exists for port %d", publicPort)))
		return nil
	}

	// 创建独立的端口代理实例
	proxy, err := ppm.createPortProxy(ctx, publicPort)
	if err != nil {
		return fmt.Errorf("failed to create port proxy: %w", err)
	}

	// 启动代理
	if err := proxy.start(); err != nil {
		return fmt.Errorf("failed to start port proxy: %w", err)
	}

	// 存储代理实例
	ppm.proxies[publicPort] = proxy

	log.Info("PortProxyManager", log.Any("Message", fmt.Sprintf("Port proxy started for port %d", publicPort)))
	return nil
}

// createPortProxy 创建端口代理实例
func (ppm *PortProxyManager) createPortProxy(ctx igoContext.IContext, publicPort int) (*PortProxy, error) {
	// 实时获取容器映射（完全依赖 port_mapping.go 的缓存机制）
	mappings, err := ppm.service.GetContainerMapping(ctx, publicPort)
	if err != nil {
		log.Error("PortProxyManager", log.Any("Error", fmt.Sprintf("Failed to get container mapping for port %d: %v", publicPort, err)))
		return nil, fmt.Errorf("failed to get container mapping: %w", err)
	}

	if len(mappings) == 0 {
		log.Warn("PortProxyManager", log.Any("Message", fmt.Sprintf("No containers found for port %d", publicPort)))
		return nil, fmt.Errorf("no containers found for port %d", publicPort)
	}

	// 创建独立的上下文
	proxyCtx, cancel := context.WithCancel(context.Background())

	proxy := &PortProxy{
		publicPort: publicPort,
		cancel:     cancel,
		ctx:        proxyCtx,
	}

	// 根据容器数量决定代理类型
	if len(mappings) == 1 {
		// 单副本：创建直接代理
		proxy.proxyType = "single"
		singleProxy, err := ppm.createSingleProxy(mappings[0])
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create single proxy: %w", err)
		}
		proxy.singleProxy = singleProxy
	} else {
		// 多副本：创建负载均衡器
		proxy.proxyType = "load_balancer"
		balancer, err := ppm.createLoadBalancer(mappings)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create load balancer: %w", err)
		}
		proxy.balancer = balancer
	}

	return proxy, nil
}

// createSingleProxy 创建单副本代理
func (ppm *PortProxyManager) createSingleProxy(mapping *ContainerMapping) (*httputil.ReverseProxy, error) {
	targetURL := fmt.Sprintf("http://localhost:%d", mapping.ContainerPort)
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("PortProxyManager", log.Any("Error", fmt.Sprintf("Proxy error for port %d -> %d: %v", mapping.ContainerPort, mapping.ContainerPort, err)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(fmt.Sprintf("Service %s is unavailable", mapping.ServiceName)))
	}

	return proxy, nil
}

// createLoadBalancer 创建负载均衡器
func (ppm *PortProxyManager) createLoadBalancer(mappings []*ContainerMapping) (*LoadBalancer, error) {
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
		backend, err := ppm.createBackend(mapping)
		if err != nil {
			log.Error("PortProxyManager", log.Any("Error", fmt.Sprintf("Failed to create backend for container %s: %v", mapping.ContainerID, err)))
			continue
		}
		balancer.backends = append(balancer.backends, backend)
	}

	if len(balancer.backends) == 0 {
		return nil, fmt.Errorf("no valid backends")
	}

	return balancer, nil
}

// createBackend 创建后端服务器
func (ppm *PortProxyManager) createBackend(mapping *ContainerMapping) (*Backend, error) {
	targetURL := fmt.Sprintf("http://localhost:%d", mapping.ContainerPort)
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("PortProxyManager", log.Any("Error", fmt.Sprintf("Backend error for container %s: %v", mapping.ContainerID, err)))
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

// start 启动端口代理
func (pp *PortProxy) start() error {
	router := gin.New()
	router.Use(gin.Recovery())

	// 根据代理类型设置路由
	if pp.proxyType == "single" {
		router.NoRoute(gin.WrapH(pp.singleProxy))
		log.Info("PortProxy", log.Any("Message", fmt.Sprintf("Starting single proxy server for port %d", pp.publicPort)))
	} else {
		router.NoRoute(func(c *gin.Context) {
			backend := pp.balancer.SelectBackend(c.Request)
			if backend == nil {
				log.Error("PortProxy", log.Any("Error", fmt.Sprintf("No available backend for port %d", pp.publicPort)))
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No available backends"})
				return
			}

			// 增加连接计数
			atomic.AddInt64(&backend.Connections, 1)
			defer atomic.AddInt64(&backend.Connections, -1)

			backend.LastUsed = time.Now()
			log.Debug("PortProxy", log.Any("Message", fmt.Sprintf("Load balancing request: %s %s -> container %d", c.Request.Method, c.Request.URL.Path, backend.ContainerMapping.ContainerPort)))

			// 代理请求
			backend.Proxy.ServeHTTP(c.Writer, c.Request)
		})
		log.Info("PortProxy", log.Any("Message", fmt.Sprintf("Starting load balancer server for port %d with %d backends", pp.publicPort, len(pp.balancer.backends))))
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", pp.publicPort),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	pp.server = server

	// 启动服务器
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("PortProxy", log.Any("Error", fmt.Sprintf("Server error for port %d: %v", pp.publicPort, err)))
		}
	}()

	return nil
}

// stop 停止端口代理
func (pp *PortProxy) stop() error {
	if pp.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := pp.server.Shutdown(ctx); err != nil {
			log.Error("PortProxy", log.Any("Error", fmt.Sprintf("Failed to shutdown server for port %d: %v", pp.publicPort, err)))
		}
	}

	// 取消上下文
	if pp.cancel != nil {
		pp.cancel()
	}

	log.Info("PortProxy", log.Any("Message", fmt.Sprintf("Port proxy stopped for port %d", pp.publicPort)))
	return nil
}

// StopPortProxy 停止端口代理
func (ppm *PortProxyManager) StopPortProxy(publicPort int) error {
	ppm.mutex.Lock()
	defer ppm.mutex.Unlock()

	proxy, exists := ppm.proxies[publicPort]
	if !exists {
		log.Info("PortProxyManager", log.Any("Message", fmt.Sprintf("No proxy found for port %d", publicPort)))
		return nil
	}

	// 停止代理
	if err := proxy.stop(); err != nil {
		return err
	}

	// 从管理器中移除
	delete(ppm.proxies, publicPort)

	log.Info("PortProxyManager", log.Any("Message", fmt.Sprintf("Port proxy stopped for port %d", publicPort)))
	return nil
}

// UpdatePortProxy 更新端口代理
func (ppm *PortProxyManager) UpdatePortProxy(ctx igoContext.IContext, publicPort int) error {
	// 先停止现有代理
	if err := ppm.StopPortProxy(publicPort); err != nil {
		log.Error("PortProxyManager", log.Any("Error", fmt.Sprintf("Failed to stop existing proxy for port %d: %v", publicPort, err)))
	}

	// 等待一小段时间确保端口释放
	time.Sleep(100 * time.Millisecond)

	// 重新启动代理
	return ppm.StartPortProxy(ctx, publicPort)
}

// GetProxyStats 获取代理统计信息
func (ppm *PortProxyManager) GetProxyStats(ctx igoContext.IContext) map[string]interface{} {
	ppm.mutex.RLock()
	defer ppm.mutex.RUnlock()

	singleCount := 0
	balancerCount := 0

	proxyDetails := make([]map[string]interface{}, 0)

	for port, proxy := range ppm.proxies {
		detail := map[string]interface{}{
			"public_port": port,
			"server_addr": fmt.Sprintf(":%d", port),
			"type":        proxy.proxyType,
		}

		if proxy.proxyType == "single" {
			singleCount++
		} else {
			balancerCount++
			if proxy.balancer != nil {
				detail["strategy"] = proxy.balancer.strategy
				detail["backend_count"] = len(proxy.balancer.backends)

				backends := make([]map[string]interface{}, 0)
				for _, backend := range proxy.balancer.backends {
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
		}

		proxyDetails = append(proxyDetails, detail)
	}

	stats := map[string]interface{}{
		"total_proxies":  len(ppm.proxies),
		"single_proxies": singleCount,
		"load_balancers": balancerCount,
		"proxy_details":  proxyDetails,
	}

	return stats
}

// Shutdown 关闭所有代理
func (ppm *PortProxyManager) Shutdown() error {
	ppm.mutex.Lock()
	defer ppm.mutex.Unlock()

	var errors []error
	for port, proxy := range ppm.proxies {
		if err := proxy.stop(); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop proxy for port %d: %w", port, err))
		}
	}

	// 清理所有代理
	ppm.proxies = make(map[int]*PortProxy)

	log.Info("PortProxyManager", log.Any("Message", "All port proxies shutdown"))

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}
	return nil
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