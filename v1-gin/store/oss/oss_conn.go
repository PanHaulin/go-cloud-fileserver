package oss

import (
	"gitee.com/porient/go-cloud/v1-gin/config"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var ossCli *oss.Client

// 创建Oss client 对象
func Client() *oss.Client {
	if ossCli != nil {
		return ossCli
	}

	sugarLogger := logger.GetLoggerOr()
	ossCli, err := oss.New(config.OSS_ENDPOINT, config.OSS_ACCESS_KEY_ID, config.OSS_ACCESS_KEY_SECRET)
	if err != nil {
		sugarLogger.Errorf("Get oss client failed, err: %s", err.Error())
		return nil
	}

	return ossCli
}

// 获取 Bucket 存储空间
func Bucket() *oss.Bucket {
	cli := Client()

	sugarLogger := logger.GetLoggerOr()
	if cli != nil {
		bucket, err := cli.Bucket(config.OSS_BUCKET)
		if err != nil {
			sugarLogger.Errorf("Get oss bucket failed, err: %s", err.Error())
			return nil
		}
		return bucket
	}

	return nil
}

// 为 Object 生成下载url
func DownloadURL(objName string) string {
	signURL, err := Bucket().SignURL(objName, oss.HTTPGet, 3600)
	sugarLogger := logger.GetLoggerOr()
	if err != nil {
		sugarLogger.Errorf("Failed to generate signurl, err: %s", err.Error())
		return ""
	}
	sugarLogger.Infof("Get downloadurl: %s", signURL)
	return signURL
}

// TODO：创建公共读Bucket，用于存放可公开的静态资源

// 对象生命周期管理，过期就删除的Bucket，存放如日志等临时文件
func BuildLifecycleRule(bucketName string) {
	// 以log为前缀的文件距最后修改日期30天后删除
	rule := oss.BuildLifecycleRuleByDays("rule1", "log/", true, 30)
	rules := []oss.LifecycleRule{rule}

	Client().SetBuckeyLifecycle(bucketName, rules)
}
