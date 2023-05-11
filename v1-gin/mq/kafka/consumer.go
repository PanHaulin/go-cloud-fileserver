package kafka

import (
	"strings"

	"gitee.com/porient/go-cloud/v1-gin/config"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"github.com/Shopify/sarama"
)

var consumeDone chan bool

// 消费消息
func StartConsume(topic string, callback func(msg []byte) bool) {
	sugarLogger := logger.GetLoggerOr()

	// 创建consumer
	consumer, err := sarama.NewConsumer(strings.Split(config.KAFKA_URLS, ","), nil)
	if err != nil {
		sugarLogger.Errorf("Fail to create consumer")
		return
	}

	partitionList, err := consumer.Partitions(topic)
	if err != nil {
		sugarLogger.Errorf("Fail to get partitions for topic: %s", topic)
		return
	}

	consumeDone = make(chan bool) // 用于中断消费过程

	go func() {
		// 遍历所有分区
		for partition := range partitionList {
			// 针对每一个分区创建一个对应的消费者
			pc, err := consumer.ConsumePartition(topic, int32(partition), sarama.OffsetNewest)
			if err != nil {
				sugarLogger.Errorf("Fail to start consumer for partition %d, err: %s", partition, err.Error())
				return
			}
			defer pc.AsyncClose()
			// 异步从每个分区消费消息
			go func(sarama.PartitionConsumer) {
				for msg := range pc.Messages() {
					success := callback(msg.Value)
					if !success {
						// 处理失败
						// TODO: 将消息重新提交
						break
					}
				}
			}(pc)
		}
	}()

	<-consumeDone
}
