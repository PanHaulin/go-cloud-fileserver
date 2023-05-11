package main

import (
	"encoding/json"
	"os"

	"gitee.com/porient/go-cloud/v1-gin/logger"
	"gitee.com/porient/go-cloud/v1-gin/meta"
	"gitee.com/porient/go-cloud/v1-gin/mq/kafka"
	"gitee.com/porient/go-cloud/v1-gin/store/oss"
)

// 处理文件转移
func ProcessTransfer(msg []byte) bool {
	sugarLogger := logger.GetLoggerOr()
	// 解析msg
	data := kafka.TransferMsg{}
	err := json.Unmarshal(msg, &data)
	if err != nil {
		sugarLogger.Errorf("msg unmarshal failed, err: %s", err.Error())
		return false
	}

	// 根据临时存储文件路径，创建文件句柄
	file, err := os.Open(data.CurLocation)
	if err != nil {
		sugarLogger.Errorf("failed to open file %s, err: %s", data.CurLocation, err.Error())
		return false
	}

	// 将文件上传至oss
	err = oss.Bucket().PutObject(data.DestLocation, file)
	if err != nil {
		sugarLogger.Errorf("Put object to oss failed, err:%s", err.Error())
		return false
	}

	// 修改文件元信息中的存储路径
	fMeta := meta.GetFileMeta(data.FileHash)
	fMeta.Location = data.DestLocation
	meta.UpdateFileMetaDB(fMeta)
	return true
}

func main() {
	sugarLogger := logger.GetLoggerOr()
	sugarLogger.Info("Start transfer files to oss...")
	kafka.StartConsume(
		"transfer_oss",
		ProcessTransfer,
	)
	sugarLogger.Info("All files transfer to oss!")
}
