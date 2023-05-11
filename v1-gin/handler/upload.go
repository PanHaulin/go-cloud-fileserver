package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"gitee.com/porient/go-cloud/v1-gin/code"
	"gitee.com/porient/go-cloud/v1-gin/common"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"gitee.com/porient/go-cloud/v1-gin/meta"
	"gitee.com/porient/go-cloud/v1-gin/mq/kafka"
	"gitee.com/porient/go-cloud/v1-gin/store/oss"
	"github.com/gin-gonic/gin"
)

// 跳转上传页面
func UploadPageHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()
	c.Redirect(http.StatusFound, "/static/view/index.html")
	sugarLogger.Info("redirect to upload page")
}

// 上传文件接口
func UploadHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()

	// 以Form的形式接收文件流并存储到本地目录
	file, head, err := c.Request.FormFile("file")
	defer file.Close()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Upload failed",
			"code": code.ParamBindError,
		})
		sugarLogger.Errorf("read file failed, err:%s", err.Error())
		return
	}

	// 创建元信息
	fileMeta := meta.FileMeta{
		FileName: head.Filename,
		Location: "/home/go-dev/go-cloud/tmp/" + head.Filename,
	}

	// 将文件流拷贝到写入文件中
	newFile, err := os.Create(fileMeta.Location)
	defer newFile.Close()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Upload failed",
			"code": code.ServerError,
		})
		sugarLogger.Errorf("create file failed, err:%s", err.Error())
		return
	}

	fileMeta.FileSize, err = io.Copy(newFile, file)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Upload failed",
			"code": code.ServerError,
		})
		sugarLogger.Errorf("save file failed, err:%s", err.Error())
		return
	}

	// 计算MD5
	newFile.Seek(0, 0)                    // 先将指针移动到开头
	fileHash := meta.HashFile(newFile)    // 计算哈希值
	meta.SetMetaHash(&fileMeta, fileHash) // 设置哈希值

	// 异步将文件写入到oss中
	newFile.Seek(0, 0) // 指针移动到开头
	ossPath := "oss/" + fileHash

	data := kafka.TransferMsg{
		FileHash:      fileHash,
		CurLocation:   fileMeta.Location,
		DestLocation:  ossPath,
		DestStoreType: common.STORE_OSS,
	}
	msg, _ := json.Marshal(data)
	success := kafka.Publish("transfer_oss", string(msg))
	if !success {
		// TODO: 重试发送
	}

	// 幂等校验，检查是否存在该文件
	_, err = meta.GetFileMetaDB(fileHash)
	if err != nil {
		// 不存在，将元信息保存到数据库
		meta.UpdateFileMetaDB(fileMeta)
	} else {
		// 否则跳过唯一文件表，只存到用户文件表中
		// 删除上传好的文件
		newFile.Close()
		os.Remove(fileMeta.Location)
	}

	// 更新用户文件表
	username := c.Request.FormValue("username")
	success = meta.OnUserFileUploadFinished(username, fileMeta.FileMD5, fileMeta.FileName)
	if success {
		// 重定向到上传成功
		c.Redirect(http.StatusFound, "/file/upload/success")
		sugarLogger.Info("redirect to upload success page")
	} else {
		// 上传失败
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Upload failed",
			"code": code.MySQLExecError,
		})
		sugarLogger.Errorf("save user-file meta failed, err:%s", err.Error())
		return
	}

}

// 上传成功接口
func UploadSuccessHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()
	c.JSON(http.StatusOK, gin.H{
		"msg":  "Upload succeeded",
		"code": 0,
	})
	sugarLogger.Info("upload file succeeded")
}

// 文件下载接口
func DownloadHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()

	filehash := c.Request.FormValue("filehash")
	fMeta := meta.GetFileMeta(filehash)
	c.Header("Content-Description", "attachment;filename=\""+fMeta.FileName+"\"")
	c.File(fMeta.Location)

	sugarLogger.Infof("download file %s succecced", fMeta.FileName)

	// 打开文件
	// f, err := os.Open(fMeta.Location)
	// if err != nil {
	// 	// 文件不存在
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	sugarLogger.Errorf("Failed to download file, err: %s", err.Error())
	// 	return
	// }
	// defer f.Close()

	// // 将文件内容加载到缓冲区
	// data, err := ioutil.ReadAll(f)
	// if err != nil {
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	sugarLogger.Errorf("Failed to load data from %s, err:%s", fMeta.Location, err.Error())
	// 	return
	// }

	// // 返回数据（文件过大时需要以流式的方式多次传输）
	// w.Header().Set("Content-Type", "application/octect-stream") // 字节流，处理方式为下载
	// w.Header().Set("Content-Description", "attachment;filename=\""+fMeta.FileName+"\"")
	// w.Write(data)

	// sugarLogger.Infof("download file %s success", fMeta.FileName)
}

// 文件删除接口
func FileDelHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()

	username := c.Request.FormValue("username")
	fileHash := c.Request.FormValue("filehash")

	// TODO: 检查文件是否被多个用户持有，如果是则只删除用户文件表下的记录
	count := meta.NumUsersFileBelongsTo(fileHash)
	switch count {
	case 0:
		// 没有找到该文件，直接返回
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Delete succeeded",
			"code": 0,
		})
		sugarLogger.Info("can not find file <%s>", fileHash)
		return
	case 1:
		// 删除用户文件表记录
		meta.DelFileOfUser(username, fileHash)
	default:
		// count > 1, 删除用户文件表记录并返回
		meta.DelFileOfUser(username, fileHash)
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Delete succeeded",
			"code": 0,
		})
		sugarLogger.Info("delete item from user files")
		return
	}

	// 删除本地文件
	fMeta := meta.GetFileMeta(fileHash)
	os.Remove(fMeta.Location) // 有可能失败

	// 删除文件元信息
	go meta.RemoveFileMeta(fileHash)

	c.JSON(http.StatusOK, gin.H{
		"msg":  "OK",
		"code": 0,
	})

	sugarLogger.Infof("delete file %s succeeded", fMeta.FileName, fileHash)
}

// 获取文件元信息接口
func GetFileMetaHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()

	// 解析参数
	fileHash := c.Request.FormValue("filehash")

	// fMeta := meta.GetFileMeta(fileHash)
	fMeta, err := meta.GetFileMetaDB(fileHash)
	if err != nil {
		// 无法查询到该记录， gorm.ErrRecordNotFound
		c.JSON(http.StatusOK, gin.H{
			"msg":  "fail to get file meta",
			"code": code.MySQLExecError,
		})
		sugarLogger.Errorf("recode of <%s> is not found, err: %s", fileHash, err.Error())
		return
	}

	// 封装响应
	c.JSON(http.StatusOK, gin.H{
		"msg":  "OK",
		"code": 0,
		"data": fMeta,
	})

	sugarLogger.Infof("get meta of <%s> succeeded", fileHash)

	// data, err := json.Marshal(fMeta)
	// if err != nil {
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	sugarLogger.Errorf("fMeta to JSON Failed, err:%s", err.Error())
	// 	return
	// }
	// w.Write(data)
	// sugarLogger.Infof("get file meta of (%s)", fileHash)
}

// 更新文件元信息接口(如重命名)
func FileUpdateMetaHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()

	opType := c.Request.FormValue("optype") // 操作类型，e.g., rename
	fileHash := c.Request.FormValue("filehash")
	curFileMeta := meta.GetFileMeta(fileHash)

	// if r.Method != "POST" {
	// 	w.WriteHeader(http.StatusMethodNotAllowed)
	// 	sugarLogger.Errorf("update file meta of (%s) witout POST", fileHash)
	// 	return
	// }

	switch opType {
	case "0":
		// 重命名
		curFileMeta.FileName = c.Request.FormValue("filename")
	default:
		// 操作非法
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Invalid operate type",
			"code": code.ParamBindError,
		})
		sugarLogger.Errorf("update file meta with unknown optype: %s", opType)
		return
	}

	// 更新元信息
	meta.UpdateFileMetaDB(curFileMeta)

	// 返回修改后的元信息
	c.JSON(http.StatusOK, gin.H{
		"msg":  "OK",
		"code": 0,
		"data": curFileMeta,
	})

	sugarLogger.Infof("update file meta of <%s>", fileHash)

	// data, err := json.Marshal(curFileMeta)
	// if err != nil {
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	sugarLogger.Errorf("fMeta to JSON failed, err: %s", err.Error())
	// }
	// w.WriteHeader(http.StatusOK)
	// w.Write(data)
	// sugarLogger.Infof("update file meta of (%s)", fileHash)

}

// 批量获取用户文件元信息接口
func FilesQueryHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()

	limitCnt, _ := strconv.Atoi(c.Request.FormValue("limit"))
	username := c.Request.FormValue("username")
	userfiles := meta.QueryUserFileMetas(username, limitCnt) // 没有返回err，已经在调用的函数里log了

	if userfiles == nil {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "fail to get user files",
			"code": code.MySQLExecError,
		})
		return
	}

	// 封装响应
	c.JSON(http.StatusOK, gin.H{
		"msg":  "OK",
		"code": 0,
		"data": userfiles,
	})
	sugarLogger.Infof("get files of user [%s] succeeded", username)

	// data, err := json.Marshal(userfiles)
	// if err != nil {
	// 	// 返回了err，需要在当前log err
	// 	sugarLogger.Errorf("Failed to parse userfiles to json, err: %s", err.Error())
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	return
	// }
	// w.Write(data)
}

// 尝试秒传接口
func TryFastUploadHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()

	username := c.Request.FormValue("username")
	filehash := c.Request.FormValue("filehash")
	filename := c.Request.FormValue("filename")

	// 从文件表中查询相同hash文件记录
	_, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		// 查不到记录
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Fail to fast upload, please try normal upload",
			"code": code.MySQLExecError,
		})
		sugarLogger.Errorf("Failed to get file meta, err: %s", err.Error())
		return
	}

	// 秒传成功则将文件信息写入用户文件表，返回成功
	success := meta.OnUserFileUploadFinished(username, filehash, filename)
	if !success {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Fail to fast upload, please retry",
			"code": code.MySQLExecError,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"msg":  "OK",
		"code": 0,
	})
	sugarLogger.Info("fast upload succeeded")
}

// 从oss返回url，减少文件传输次数
func DownloadURLHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()

	filehash := c.Request.FormValue("filehash")
	// 从文件表查找oss地址
	fMeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Fail to get downloadurl",
			"code": code.MySQLExecError,
		})
		sugarLogger.Errorf("filehash is not found, err: %s", err.Error())
	}

	signURL := oss.DownloadURL(fMeta.Location)
	c.JSON(http.StatusOK, gin.H{
		"msg":  "OK",
		"code": 0,
		"url":  signURL,
	})
	sugarLogger.Info("get download url from oss succeeded")
}
