package main

import (
	"gitee.com/porient/go-cloud/v1-gin/config"
	"gitee.com/porient/go-cloud/v1-gin/db/mysql"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"gitee.com/porient/go-cloud/v1-gin/meta"
	"gitee.com/porient/go-cloud/v1-gin/route"
)

func main() {
	// 迁移数据库schema
	db := mysql.DBConn()
	db.AutoMigrate(&meta.FileMeta{}, &meta.User{}, &meta.UserToken{}, &meta.UserFile{})

	sugarLogger := logger.GetLoggerOr()
	sugarLogger.Sync()

	router := route.Router()
	router.Run(config.UPLOAD_SERVICE_HOST)

	defer sugarLogger.Info("upload service finished")
}
