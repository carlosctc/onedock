package api

import (
	"time"

	"github.com/aichy126/onedock/service"
	"github.com/aichy126/onedock/utils"
	"github.com/gin-gonic/gin"
)

// Api
type Api struct {
	ser *service.Service
}

// NewApi
func NewApi() *Api {
	return &Api{
		ser: service.NewService(),
	}
}

// @Summary 健康检查
// @Description 用于检查 OneDock 服务的健康状态和连通性，返回服务状态信息
// @Tags 系统监控
// @Accept  json
// @Produce  json
// @Router /onedock/ping [get]
// @Success 200 {object} object{code=int,data=object,msg=string} "服务正常运行"
func (s *Api) Ping(c *gin.Context) {
	Body, _ := c.GetRawData()
	now := time.Now().Format("2006.01.02 03:04:05")
	utils.Rsucc(c, gin.H{
		"message": "pong",
		"time":    now,
		"body":    string(Body),
		"header":  c.Request.Header,
	})
}
