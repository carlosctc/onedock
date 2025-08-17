package utils

import (
	"net/http"
	"os"
	"strconv"

	"github.com/aichy126/igo"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Rfail 错误返回
func Rfail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"msg":  msg,
		"data": nil,
	})
}

// Rsucc 成功返回
func Rsucc(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": data,
		"msg":  "succeed",
	})
}

func RHtml(c *gin.Context, html string, data interface{}) {
	c.HTML(http.StatusOK, html, gin.H{
		"dev":  os.Getenv("DEVCODE"),
		"data": data,
	})
}

func StringToInt(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return num
}
func StringToInt64(str string) int64 {
	num, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0
	}
	return num
}

func ConfGetbool(path string) bool {
	return igo.App.Conf.GetBool(path)
}

func ConfGetString(path string) string {
	return igo.App.Conf.GetString(path)
}

func ConfGetInt64(path string) int64 {
	return igo.App.Conf.GetInt64(path)
}

func ConfGetInt(path string) int {
	return igo.App.Conf.GetInt(path)
}

func GenerateToken() string {
	uid, _ := uuid.NewUUID()
	return uid.String()
}
