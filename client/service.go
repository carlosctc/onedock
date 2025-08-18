package onedockclient

import (
	"fmt"
)

// Ping 健康检查
func (c *Client) Ping() (*PingResponse, error) {
	resp, err := c.doRequest("GET", "/onedock/ping", nil)
	if err != nil {
		return nil, NewNetworkError(err)
	}

	var result PingResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeployService 部署新服务
func (c *Client) DeployService(req *ServiceRequest) (*Service, error) {
	if err := c.validateServiceRequest(req); err != nil {
		return nil, err
	}

	resp, err := c.doRequest("POST", "/onedock/", req)
	if err != nil {
		return nil, NewNetworkError(err)
	}

	var result Service
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListServices 获取所有服务列表
func (c *Client) ListServices() (*ServiceListResponse, error) {
	resp, err := c.doRequest("GET", "/onedock/", nil)
	if err != nil {
		return nil, NewNetworkError(err)
	}

	result := new(ServiceListResponse)
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetService 获取指定服务信息
func (c *Client) GetService(name string) (*Service, error) {
	if name == "" {
		return nil, NewValidationError("name", "service name cannot be empty")
	}

	endpoint := fmt.Sprintf("/onedock/%s", name)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, NewNetworkError(err)
	}

	var result Service
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteService 删除指定服务
func (c *Client) DeleteService(name string) error {
	if name == "" {
		return NewValidationError("name", "service name cannot be empty")
	}

	endpoint := fmt.Sprintf("/onedock/%s", name)
	resp, err := c.doRequest("DELETE", endpoint, nil)
	if err != nil {
		return NewNetworkError(err)
	}

	return c.parseResponse(resp, nil)
}

// GetServiceStatus 获取服务详细状态
func (c *Client) GetServiceStatus(name string) (*ServiceStatusResponse, error) {
	if name == "" {
		return nil, NewValidationError("name", "service name cannot be empty")
	}

	endpoint := fmt.Sprintf("/onedock/%s/status", name)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, NewNetworkError(err)
	}

	var result ServiceStatusResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ScaleService 扩缩容服务
func (c *Client) ScaleService(name string, replicas int) error {
	if name == "" {
		return NewValidationError("name", "service name cannot be empty")
	}
	if replicas < 0 {
		return NewValidationError("replicas", "replicas must be non-negative")
	}

	endpoint := fmt.Sprintf("/onedock/%s/scale", name)
	req := &ScaleRequest{
		Replicas: replicas,
	}

	resp, err := c.doRequest("POST", endpoint, req)
	if err != nil {
		return NewNetworkError(err)
	}

	return c.parseResponse(resp, nil)
}

// GetProxyStats 获取代理统计信息
func (c *Client) GetProxyStats() (*ProxyStats, error) {
	resp, err := c.doRequest("GET", "/onedock/proxy/stats", nil)
	if err != nil {
		return nil, NewNetworkError(err)
	}

	var result ProxyStats
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// StopService 停止服务（缩容到0）
func (c *Client) StopService(name string) error {
	return c.ScaleService(name, 0)
}

// StartService 启动服务（如果已停止，恢复到1个副本）
func (c *Client) StartService(name string) error {
	return c.ScaleService(name, 1)
}

// validateServiceRequest 验证服务请求参数
func (c *Client) validateServiceRequest(req *ServiceRequest) error {
	if req == nil {
		return NewValidationError("request", "service request cannot be nil")
	}

	if req.Name == "" {
		return NewValidationError("name", "service name is required")
	}

	if req.Image == "" {
		return NewValidationError("image", "docker image is required")
	}

	if req.Tag == "" {
		return NewValidationError("tag", "image tag is required")
	}

	if req.InternalPort <= 0 {
		return NewValidationError("internal_port", "internal port must be positive")
	}

	if req.Replicas < 0 {
		return NewValidationError("replicas", "replicas must be non-negative")
	}

	if req.PublicPort < 0 {
		return NewValidationError("public_port", "public port must be non-negative")
	}

	return nil
}
