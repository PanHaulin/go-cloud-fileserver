package kafka

import (
	"strings"
	"time"

	"gitee.com/porient/go-cloud/v1-gin/config"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"github.com/Shopify/sarama"
)

var producer sarama.SyncProducer

// 初始化生产者
func InitProducer() bool {
	var err error
	sugarLogger := logger.GetLoggerOr()

	addressList := config.KAFKA_URLS

	// 设置配置
	mqConfig := sarama.NewConfig()
	// 设置producer
	// 发送完数据需要leader和follow都确认
	mqConfig.Producer.RequiredAcks = sarama.WaitForAll
	// Partition选择随机
	mqConfig.Producer.Partitioner = sarama.NewRandomPartitioner
	// 成功交付的消息将在success channel返回
	mqConfig.Producer.Return.Successes = true

	// 创建client
	kafkaClient, err := sarama.NewClient(strings.Split(addressList, ","), mqConfig)
	if err != nil {
		sugarLogger.Errorf("Failed to new kafka client, err:%s", err.Error())
		return false
	}

	// 初始化生产者
	producer, err = sarama.NewSyncProducerFromClient(kafkaClient)
	if err != nil {
		sugarLogger.Errorf("Failed to new kafka producer, err:%s", err.Error())
		return false
	}
	return true
}

// 发布消息
func Publish(topic, content string) bool {
	sugarLogger := logger.GetLoggerOr()

	// 构造消息
	msg := &sarama.ProducerMessage{
		Topic:     topic,
		Value:     sarama.StringEncoder(content),
		Timestamp: time.Now(),
	}

	// 发送消息
	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		sugarLogger.Errorf("Kafka send msg failed, err: %s", err.Error())
		return false
	}
	sugarLogger.Infof("Kafka send msg, return: t=%s, p=%s, o=%s", topic, partition, offset)
	return true
}
