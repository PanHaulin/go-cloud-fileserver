package redis

import (
	"time"

	"gitee.com/porient/go-cloud/v1-gin/config"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"github.com/go-redis/redis"
)

// go-redis操作：https://juejin.cn/post/7027347979065360392
var redisDB *redis.Client

// 获取
func RedisDBConn() *redis.Client {
	return redisDB
}

// 初始化client
func init() {
	sugarLogger := logger.GetLoggerOr()
	redisDB = redis.NewClient(&redis.Options{
		Addr:         config.REDIS_HOST,
		Password:     config.REDIS_PASSWD,
		DB:           0,
		PoolSize:     50,                // 最大连接数
		MinIdleConns: 20,                // 最小空闲连接数
		IdleTimeout:  300 * time.Second, // 空闲连接检测
	})

	// 检测连接是否成功
	_, err := redisDB.Ping().Result()
	if err != nil {
		sugarLogger.Errorf("Failed to connect Redis, err: %s", err.Error())
	}
}
