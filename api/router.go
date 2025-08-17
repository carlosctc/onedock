package api

import (
	"github.com/aichy126/onedock/middleware"
	"github.com/gin-gonic/gin"
)

func Router(r *gin.Engine) {
	r.Use(middleware.Cors())
	api := NewApi()

	r.GET("/onedock/ping", api.Ping)
	r.POST("/onedock/ping", api.Ping)

	services := r.Group("/onedock")
	services.POST("/", api.DeployOrUpdateService)       // 部署或更新服务
	services.GET("/", api.ListServices)                 // 列出所有服务
	services.GET("/:name", api.GetService)              // 获取服务
	services.DELETE("/:name", api.DeleteService)        // 删除服务
	services.GET("/:name/status", api.GetServiceStatus) // 获取服务状态
	services.POST("/:name/scale", api.ScaleService)     // 服务扩缩容
	services.GET("/proxy/stats", api.GetProxyStats)     // 获取代理统计信息
}
