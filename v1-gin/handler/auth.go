package handler

import (
	"net/http"

	"gitee.com/porient/go-cloud/v1-gin/code"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"gitee.com/porient/go-cloud/v1-gin/meta"
	"github.com/gin-gonic/gin"
)

// 鉴权中间件
func AccessAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		sugarLogger := logger.GetLoggerOr()

		// 请求前鉴权
		username := c.Request.FormValue("username")
		token := c.Request.FormValue("token")

		// 用户名或token无效
		if len(username) < 3 || !meta.IsTokenValid(token) {
			c.Abort() // 截断链路
			c.JSON(http.StatusOK, gin.H{
				"msg":  "Invalid username/token",
				"code": code.ParamBindError,
			})
			sugarLogger.Error("Invalid username or token")
			return
		}
		sugarLogger.Info("authorize succeeded")

		c.Next()

		// 请求后无操作
	}
}
