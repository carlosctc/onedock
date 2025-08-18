package middleware

import (
	"strings"

	"github.com/aichy126/igo/util"
	"github.com/aichy126/onedock/utils"
	"github.com/gin-gonic/gin"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否启用权限验证
		if !utils.ConfGetbool("auth.enabled") {
			c.Next()
			return
		}

		// 检查是否在白名单中
		path := c.Request.URL.Path
		whitelistPaths := getWhitelistPaths()
		for _, whitePath := range whitelistPaths {
			if path == whitePath {
				c.Next()
				return
			}
		}

		// 从多个位置获取 token
		token := extractToken(c)
		if token == "" {
			utils.Rfail(c, "权限验证失败：缺少访问令牌")
			c.Abort()
			return
		}

		// 验证 token
		if !isValidToken(token) {
			utils.Rfail(c, "权限验证失败：无效的访问令牌")
			c.Abort()
			return
		}

		c.Next()
	}
}

// extractToken 从请求中提取 token
// 支持多种方式：Authorization Bearer、Token Header、Query 参数
func extractToken(c *gin.Context) string {
	// 1. 从 Authorization Header 获取 Bearer Token
	auth := c.GetHeader("Authorization")
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// 2. 从 Token Header 获取
	token := c.GetHeader("Token")
	if token != "" {
		return token
	}

	// 3. 从 Query 参数获取
	token = c.Query("token")
	if token != "" {
		return token
	}

	return ""
}

// isValidToken 验证 token 是否有效
func isValidToken(token string) bool {
	validTokens := getValidTokens()
	for _, validToken := range validTokens {
		if token == validToken {
			return true
		}
	}
	return false
}

// getValidTokens 从配置中获取有效的 token 列表
func getValidTokens() []string {
	// 直接获取 tokens 数组
	tokens := util.ConfGetStringSlice("auth.tokens")

	// 如果数组获取成功且不为空，直接返回
	if len(tokens) > 0 {
		return tokens
	}

	return []string{}
}

// getWhitelistPaths 从配置中获取白名单路径
func getWhitelistPaths() []string {
	// 尝试获取白名单路径数组
	paths := util.ConfGetStringSlice("auth.whitelist_paths")

	// 如果配置了白名单路径，返回配置的路径
	if len(paths) > 0 {
		return paths
	}

	// 默认白名单（ping 接口用于健康检查）
	return []string{"/onedock/ping"}
}
