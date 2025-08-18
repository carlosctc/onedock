package main

import (
	"fmt"

	"github.com/aichy126/igo"
	"github.com/aichy126/onedock/api"
	"github.com/aichy126/onedock/docs"
	"github.com/aichy126/onedock/utils"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter the token with the `Bearer: ` prefix, e.g. "Bearer abcde12345".
//
// @securityDefinitions.apikey TokenAuth
// @in header
// @name Token
// @description Direct token in header without prefix.
//
// @securityDefinitions.apikey QueryAuth
// @in query
// @name token
// @description Token as query parameter.
func main() {
	igo.App = igo.NewApp("")
	api.Router(igo.App.Web.Router)

	//swagger
	swaggerShow := utils.ConfGetbool("swaggerui.show")
	if swaggerShow {
		InitSwaggerDocs()
		urlfmt := fmt.Sprintf("%s://%s%s/swagger/doc.json", utils.ConfGetString("swaggerui.protocol"), utils.ConfGetString("swaggerui.host"), utils.ConfGetString("swaggerui.address"))
		igo.App.Web.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL(urlfmt)))
	}

	igo.App.Web.Run()
}

// 加载执行程序
// go install github.com/swaggo/swag/cmd/swag@latest
// go get -u github.com/swaggo/gin-swagger
// go get -u github.com/swaggo/files
// swag init
func InitSwaggerDocs() {
	docs.SwaggerInfo.Title = "OneDock API"
	docs.SwaggerInfo.Description = "OneDock 是一个基于 Go 和 Gin 框架构建的 Docker 容器编排服务，提供智能端口代理和负载均衡功能。支持服务部署、扩缩容、滚动更新等容器生命周期管理操作。"
	docs.SwaggerInfo.Version = "v1.0.0"
	docs.SwaggerInfo.Host = fmt.Sprintf("%s%s", utils.ConfGetString("swaggerui.host"), utils.ConfGetString("swaggerui.address"))
	docs.SwaggerInfo.BasePath = "/"
}
