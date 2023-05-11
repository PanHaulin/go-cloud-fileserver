package handler

import (
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"gitee.com/porient/go-cloud/v1-gin/cache/redis"
	"gitee.com/porient/go-cloud/v1-gin/code"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"gitee.com/porient/go-cloud/v1-gin/meta"
	"github.com/gin-gonic/gin"
)

type MulltipartUploadInfo struct {
	FileHash   string
	FileSize   int
	UploadID   string // 上传分块ID
	ChunkSize  int    // 分块大小
	ChunkCount int    // 分块数量，向上取整
	Finished   int    // 是否已上传并合并分片
}

// 初始化分块信息接口
func InitiateMultipartUploadHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()

	// 解析用户请求参数
	username := c.Request.FormValue("username")
	filehash := c.Request.FormValue("filehash")
	filesize, err := strconv.Atoi(c.Request.FormValue("filesize"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "fail to initialize multipart info",
			"code": code.ParamBindError,
		})
		sugarLogger.Error("invalid params for initialize multipart info")
		return
	}

	// 获取redis连接
	redisDB := redis.RedisDBConn()

	// 生成分块上传信息
	upInfo := MulltipartUploadInfo{
		FileHash:   filehash,
		FileSize:   filesize,
		UploadID:   username + fmt.Sprintf("%x", time.Now().UnixNano()),
		ChunkSize:  5 * 1024 * 1024, // 5MB
		ChunkCount: int(math.Ceil(float64(filesize) / (5 * 1024 * 1024))),
	}

	// 缓存分块信息
	infoData := make(map[string]any)
	infoData["chunkcount"] = upInfo.ChunkCount
	infoData["filehash"] = upInfo.FileHash
	infoData["filesize"] = upInfo.FileSize
	err = redisDB.HMSet("MP_"+upInfo.UploadID, infoData).Err()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "fail to initialize multipart info",
			"code": code.CacheSetError,
		})
		sugarLogger.Error("Failed to hset chunk info to redis")
		return
	}

	// 将初始化数据封装为响应
	c.JSON(http.StatusOK, gin.H{
		"msg":  "OK",
		"code": 0,
		"data": upInfo,
	})
}

// 上传分块接口
func UploadPartHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()
	// 解析用户参数
	// username := c.Request.FormValue("username")
	uploadID := c.Request.FormValue("uploadid") // id是唯一的，用于区分文件
	chunkIndex := c.Request.FormValue("index")  // index 用于合并文件的不同分块
	isFinal := c.Request.FormValue("isfinal")   // 用于判断
	// chunkHash := c.Request.FormValue("chunkhash") // 分块哈希，用于校验

	// 获得redis连接
	redisDB := redis.RedisDBConn()

	// 获得文件句柄用于存储分块内容
	fpath := "/home/go-dev/go-cloud/tmp/"
	os.MkdirAll(path.Dir(fpath), 0744) // 创建目录并设置权限
	fd, err := os.Create(fpath + uploadID + "/" + chunkIndex)
	// fd, err := os.Create("/data/" + uploadID + "/" + chunkIndex)
	if err != nil {
		// 可能是权限等问题
		c.JSON(http.StatusOK, gin.H{
			"msg":  "fail to upload part",
			"code": code.ServerError,
		})
		sugarLogger.Errorf("Failed to Create chunk file, err:%s", err.Error())
		return
	}
	defer fd.Close()

	// 每次读取1M
	buf := make([]byte, 1024*1024)
	for {
		n, err := c.Request.Body.Read(buf)
		fd.Write(buf[:n])
		if err != nil {
			// EOF
			break
		}
	}
	// TODO: 获得完整分块后可以再校验一次

	// 更新redis缓存状态
	redisDB.HSet("MP_"+uploadID, "chunkidx_"+chunkIndex, 1) // 表示该分块是否上传成功

	// 返回处理结果给客户端
	c.JSON(http.StatusOK, gin.H{
		"msg":  "OK",
		"code": 0,
	})
	sugarLogger.Infof("ChunkIdx %d of UploadID %s saved", chunkIndex, uploadID)
}

// 通知上传合并接口
func CompleteUploadPartHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()
	// 解析请求参数
	upid := c.Request.FormValue("uploadid")
	username := c.Request.FormValue("username")
	filehash := c.Request.FormValue("filehash")
	filesize := c.Request.FormValue("filesize")
	filename := c.Request.FormValue("filename")

	// 检查hash是否存在于mysql
	_, err := meta.GetFileMetaDB(filehash)
	if err == nil {
		// hash存在，直接秒合并
		c.JSON(http.StatusOK, gin.H{
			"msg":  "OK",
			"code": 0,
		})
	}

	// 获得redis连接
	redisDB := redis.RedisDBConn()

	// 检查是否所有分块上传完成
	data, err := redisDB.HGetAll("MP_" + upid).Result()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "fail to merge uploaded parts",
			"code": code.CacheGetError,
		})
		sugarLogger.Errorf("redis HGetAll failed, err:%s", err.Error())
		return
	}

	totalCount, chunkCount := 0, 0
	for field, val := range data {
		if field == "chunkcount" {
			// 分块初始化信息
			totalCount, _ = strconv.Atoi(val)
		} else if strings.HasPrefix(field, "chunkidx_") && val == "1" {
			// 统计已上传分块
			chunkCount++
		}
	}
	if totalCount != chunkCount {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "fail to merge uploaded parts",
			"code": code.CacheNotExist,
		})
		sugarLogger.Errorf("want %d chunks, but gpt %d chunks", totalCount, chunkCount)
		return
	}

	// 合并分块
	fpath := "/home/go-dev/go-cloud/tmp/"
	mfd, err := os.OpenFile(fpath+filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "fail to merge uploaded parts",
			"code": code.ServerError,
		})
		sugarLogger.Errorf("fail to open file, err: %s", err.Error())
		return
	}
	mfd.Close()

	// chunkidx 从1开始
	for i := 1; i <= totalCount; i++ {
		f, err := os.OpenFile(fpath+upid+"/"+strconv.Itoa(i)+"/"+filename, os.O_RDONLY, os.ModePerm)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"msg":  "fail to merge uploaded parts",
				"code": code.ServerError,
			})
			sugarLogger.Errorf("fail to open chunk file, err: %s", err.Error())
			return
		}

		b, err := ioutil.ReadAll(f)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"msg":  "fail to merge uploaded parts",
				"code": code.ServerError,
			})
			sugarLogger.Errorf("fail to read chunk file, err: %s", err.Error())
			return
		}

		mfd.Write(b)
		f.Close()
	}

	// 生成文件元信息并校验哈希值
	stat, _ := mfd.Stat()
	fileMeta := meta.FileMeta{ // 创建元信息
		FileName: filename,
		Location: fpath + filename,
		FileSize: stat.Size(),
	}

	fs, _ := strconv.Atoi(filesize)
	ifs := int64(fs)
	if ifs != fileMeta.FileSize {
		// 文件大小对不上，跳过hash校验
		c.JSON(http.StatusOK, gin.H{
			"msg":  "fail to merge uploaded parts",
			"code": code.ParamBindError,
		})
		sugarLogger.Errorf("Complete upload failed: want filesize %d, but got %d", ifs, fileMeta.FileSize)
		return
	}

	mfd.Seek(0, 0)                        // 先将指针移动到开头
	fileHash := meta.HashFile(mfd)        // 计算哈希值
	meta.SetMetaHash(&fileMeta, fileHash) // 设置哈希值

	if filehash != fileHash {
		// 哈希校验失败
		c.JSON(http.StatusOK, gin.H{
			"msg":  "fail to merge uploaded parts",
			"code": code.ParamBindError,
		})
		sugarLogger.Errorf("Complete upload failed: want filehash %s, but got %s", filehash, fileHash)
		return
	}

	success := meta.UpdateFileMetaDB(fileMeta) // 更新唯一文件表
	if !success {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "fail to merge uploaded parts",
			"code": code.MySQLExecError,
		})
		return
	}

	// 更新用户文件表
	success = meta.OnUserFileUploadFinished(username, fileMeta.FileMD5, fileMeta.FileName)
	if !success {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "fail to merge uploaded parts",
			"code": code.MySQLExecError,
		})
		return
	}

	// 删除分块文件及redis分块缓存（可以改为异步)
	os.RemoveAll(fpath + upid + "/")
	for chunkIndex := 1; chunkIndex <= totalCount; chunkIndex++ {
		redisDB.HDel("MP_"+upid, "chunkidx_"+strconv.Itoa(chunkIndex))
	}

	// 响应处理结果
	c.JSON(http.StatusOK, gin.H{
		"msg":  "OK",
		"code": 0,
	})

	sugarLogger.Info("Merge file succeeded")
}

// 取消上传分块接口
func CancelUploadPartHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()
	// 获取参数
	uploadID := c.Request.FormValue("uploadid")

	// 删除已存在的分块文件
	fpath := "/home/go-dev/go-cloud/tmp/"
	os.RemoveAll(fpath + uploadID + "/")

	// 统计分块文件
	delFields := []string{}
	redisDB := redis.RedisDBConn()
	data, _ := redisDB.HGetAll("MP_" + uploadID).Result()
	for field := range data {
		if strings.HasPrefix(field, "chunkidx_") {
			// 记录已上传分块
			delFields = append(delFields, field)
		}
	}
	redisDB.HDel("MP_"+uploadID, delFields...)

	sugarLogger.Info("cancel multipart upload")
	// TODO: 更新用户历史操作
}

// 查看分块上传的整体状态接口
func MultipartUploadStatusHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()
	// 解析参数
	uploadID := c.Request.FormValue("uploadid")

	// 获取分块初始化信息
	// 获得redis连接
	redisDB := redis.RedisDBConn()

	// 获取已上传的分块信息
	data, err := redisDB.HGetAll("MP_" + uploadID).Result()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "fail to check upload status",
			"code": code.CacheNotExist,
		})
		sugarLogger.Error("invalid uploadid, err: %s", err.Error())
		return
	}

	totalCount, chunkCount := 0, 0
	for field, val := range data {
		if field == "chunkcount" {
			// 分块初始化信息
			totalCount, _ = strconv.Atoi(val)
		} else if strings.HasPrefix(field, "chunkidx_") && val == "1" {
			// 统计已上传分块
			chunkCount++
		} else if field == "finished" && val == "1" {
			break // 已经上传完成，直接返回
		}
	}

	// TODO: 查询到完成状态后直接返回
	percent := int(float64(chunkCount) / float64(totalCount) * 100)
	c.JSON(http.StatusOK, gin.H{
		"msg":  "OK",
		"code": 0,
		"data": percent,
	})

	// TODO: 如果分片全部到达但还没有合并，则触发合并

	sugarLogger.Info("query upload status succeeded")
}
