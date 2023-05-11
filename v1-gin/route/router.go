package route

import (
	"gitee.com/porient/go-cloud/v1-gin/handler"
	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	// 类似于 http.ServeMux
	router := gin.Default()

	// 处理静态资源
	router.Static("/static/", "./static")

	// 用户服务，无需权限校验
	router.GET("/user/signup", handler.SignupPageHandler) // 跳转注册页面
	router.POST("/user/signup", handler.SignupHandler)    // 注册
	router.GET("/user/signin", handler.SigninPageHandler) // 跳转登录页面
	router.POST("/user/signin", handler.SigninHandler)    // 登录

	// 权限验证中间件，Use之后注册的路由才会生效
	router.Use(handler.AccessAuth())

	// 文件服务
	router.GET("/file/upload")
	router.POST("/file/upload", handler.UploadHandler)               // 普通上传
	router.GET("/file/upload/success", handler.UploadSuccessHandler) // 上传成功
	router.POST("/file/meta", handler.GetFileMetaHandler)            // 获取文件元信息
	router.POST("/file/download", handler.DownloadHandler)           // 下载文件
	router.POST("/file/meta/update", handler.FileUpdateMetaHandler)  // 更新文件元信息
	router.POST("/file/delete", handler.FileDelHandler)              // 删除文件
	router.POST("/file/fastupload", handler.TryFastUploadHandler)    // 秒传
	router.POST("/file/downloadurl", handler.DownloadURLHandler)     // 获取下载url

	// 分块上传
	router.POST("/file/mpupload/init", handler.InitiateMultipartUploadHandler) // 初始化分块信息
	router.POST("/file/mpupload/uppart", handler.UploadPartHandler)            // 上传分块
	router.POST("/file/mpupload/complete", handler.CompleteUploadPartHandler)  // 通知分块上传完成
	router.POST("/file/mpupload/cancel", handler.CancelUploadPartHandler)      // 取消上传分块
	router.POST("/file/mpupload/status", handler.MultipartUploadStatusHandler) // 查看分块上传的整体状态

	return router
}
