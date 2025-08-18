package api

import (
	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/log"
	"github.com/aichy126/onedock/models"
	"github.com/aichy126/onedock/utils"
	"github.com/gin-gonic/gin"
)

// DeployOrUpdateService 部署或更新服务
// @Summary 部署或更新服务
// @Description 部署新的服务或更新现有服务配置，支持容器镜像、端口映射、环境变量、卷挂载等完整配置
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param service body models.ServiceRequest true "服务配置信息"
// @Success 200 {object} object{code=int,data=models.Service,msg=string} "部署成功"
// @Failure 400 {object} object{code=int,msg=string,data=object} "请求参数错误"
// @Failure 401 {object} object{code=int,msg=string,data=object} "权限验证失败"
// @Failure 500 {object} object{code=int,msg=string,data=object} "服务器内部错误"
// @Security BearerAuth || TokenAuth || QueryAuth
// @Router /onedock [post]
func (api *Api) DeployOrUpdateService(c *gin.Context) {
	var req models.ServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("API", log.Any("Error", err), log.Any("Message", "无效的请求参数"))
		utils.Rfail(c, "invalid request body: "+err.Error())
		return
	}

	// 验证必填参数
	if req.Name == "" || req.Image == "" || req.Tag == "" || req.InternalPort <= 0 {
		utils.Rfail(c, "missing required fields: name, image, tag, internal_port")
		return
	}
	ctx := context.Ginform(c)
	// 调用服务层
	service, err := api.ser.DeployOrUpdateService(ctx, &req)
	if err != nil {
		log.Error("API", log.Any("Error", err), log.Any("ServiceName", req.Name), log.Any("Message", "部署服务失败"))
		utils.Rfail(c, err.Error())
		return
	}
	utils.Rsucc(c, service)
}

// ListServices 列出所有服务
// @Summary 列出所有服务
// @Description 获取系统中所有部署的服务列表，包括服务基本信息、状态和副本数量
// @Tags 服务管理
// @Accept json
// @Produce json
// @Success 200 {object} object{code=int,data=object{Services=[]models.Service,Total=int},msg=string} "获取成功"
// @Failure 401 {object} object{code=int,msg=string,data=object} "权限验证失败"
// @Security BearerAuth || TokenAuth || QueryAuth
// @Router /onedock [get]
func (api *Api) ListServices(c *gin.Context) {
	ctx := context.Ginform(c)
	services := api.ser.ListServices(ctx)

	// 转换为值类型切片
	serviceList := make([]models.Service, len(services))
	for i, service := range services {
		serviceList[i] = *service
	}
	utils.Rsucc(c, gin.H{
		"Services": serviceList,
		"Total":    len(services),
	})
}

// GetService 获取服务详情
// @Summary 获取指定服务详情
// @Description 根据服务名称获取服务的详细信息，包括配置、状态等
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param name path string true "服务名称" example:"nginx-web"
// @Success 200 {object} object{code=int,data=models.Service,msg=string} "获取成功"
// @Failure 400 {object} object{code=int,msg=string,data=object} "请求参数错误"
// @Failure 401 {object} object{code=int,msg=string,data=object} "权限验证失败"
// @Failure 404 {object} object{code=int,msg=string,data=object} "服务未找到"
// @Security BearerAuth || TokenAuth || QueryAuth
// @Router /onedock/{name} [get]
func (api *Api) GetService(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		utils.Rfail(c, "service name is required")
		return
	}
	ctx := context.Ginform(c)
	service := api.ser.GetService(ctx, name)
	if service == nil {
		utils.Rfail(c, "service not found")
		return
	}
	utils.Rsucc(c, service)
}

// DeleteService 删除服务
// @Summary 删除指定服务
// @Description 删除指定的服务及其所有相关容器和资源，操作不可逆
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param name path string true "服务名称" example:"nginx-web"
// @Success 200 {object} object{code=int,data=object,msg=string} "删除成功"
// @Failure 400 {object} object{code=int,msg=string,data=object} "请求参数错误"
// @Failure 401 {object} object{code=int,msg=string,data=object} "权限验证失败"
// @Failure 500 {object} object{code=int,msg=string,data=object} "服务器内部错误"
// @Security BearerAuth || TokenAuth || QueryAuth
// @Router /onedock/{name} [delete]
func (api *Api) DeleteService(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		utils.Rfail(c, "service name is required")
		return
	}
	ctx := context.Ginform(c)

	err := api.ser.DeleteService(ctx, name)
	if err != nil {
		log.Error("API", log.Any("Error", err), log.Any("ServiceName", name), log.Any("Message", "删除服务失败"))
		utils.Rfail(c, err.Error())
		return
	}
	utils.Rsucc(c, gin.H{})
}

// GetServiceStatus 获取服务状态
// @Summary 获取服务运行状态
// @Description 获取指定服务的详细运行状态，包括副本信息、健康状态、实例详情等
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param name path string true "服务名称" example:"nginx-web"
// @Success 200 {object} object{code=int,data=models.ServiceStatusResponse,msg=string} "获取成功"
// @Failure 400 {object} object{code=int,msg=string,data=object} "请求参数错误"
// @Failure 401 {object} object{code=int,msg=string,data=object} "权限验证失败"
// @Failure 404 {object} object{code=int,msg=string,data=object} "服务未找到"
// @Security BearerAuth || TokenAuth || QueryAuth
// @Router /onedock/{name}/status [get]
func (api *Api) GetServiceStatus(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		utils.Rfail(c, "service name is required")
		return
	}
	ctx := context.Ginform(c)
	status, err := api.ser.GetServiceStatus(ctx, name)
	if err != nil {
		log.Error("API", log.Any("Error", err), log.Any("ServiceName", name), log.Any("Message", "获取服务状态失败"))
		utils.Rfail(c, err.Error())
		return
	}
	utils.Rsucc(c, status)
}

// ScaleService 服务扩缩容
// @Summary 服务扩缩容
// @Description 调整指定服务的副本数量，支持扩容和缩容操作，实际创建或删除容器实例
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param name path string true "服务名称" example:"nginx-web"
// @Param scale body models.ScaleRequest true "扩缩容配置"
// @Success 200 {object} object{code=int,data=object,msg=string} "扩缩容成功"
// @Failure 400 {object} object{code=int,msg=string,data=object} "请求参数错误"
// @Failure 401 {object} object{code=int,msg=string,data=object} "权限验证失败"
// @Failure 500 {object} object{code=int,msg=string,data=object} "服务器内部错误"
// @Security BearerAuth || TokenAuth || QueryAuth
// @Router /onedock/{name}/scale [post]
func (api *Api) ScaleService(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		utils.Rfail(c, "service name is required")
		return
	}

	var req models.ScaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("API", log.Any("Error", err), log.Any("Message", "无效的请求参数"))
		utils.Rfail(c, "invalid request body: "+err.Error())
		return
	}

	// 验证副本数
	if req.Replicas < 0 {
		utils.Rfail(c, "replicas must be greater than or equal to 0")
		return
	}
	ctx := context.Ginform(c)
	err := api.ser.ScaleService(ctx, name, req.Replicas)
	if err != nil {
		log.Error("API", log.Any("Error", err), log.Any("ServiceName", name), log.Any("Replicas", req.Replicas), log.Any("Message", "扩缩容失败"))
		utils.Rfail(c, err.Error())
		return
	}
	utils.Rsucc(c, gin.H{
		"service":  name,
		"replicas": req.Replicas,
	})
}

// GetProxyStats 获取代理统计信息
// @Summary 获取端口代理统计信息
// @Description 获取所有端口代理的统计信息，包括单副本代理和负载均衡器的详细状态
// @Tags 服务管理
// @Accept json
// @Produce json
// @Success 200 {object} object{code=int,data=object,msg=string} "获取成功"
// @Failure 401 {object} object{code=int,msg=string,data=object} "权限验证失败"
// @Security BearerAuth || TokenAuth || QueryAuth
// @Router /onedock/proxy/stats [get]
func (api *Api) GetProxyStats(c *gin.Context) {
	ctx := context.Ginform(c)
	stats := api.ser.PortManager.GetProxyStats(ctx)
	utils.Rsucc(c, stats)
}
