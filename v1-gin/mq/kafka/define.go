package kafka

import "gitee.com/porient/go-cloud/v1-gin/common"

type TransferMsg struct {
	FileHash      string           // 文件哈希
	CurLocation   string           // 当前文件位置
	DestLocation  string           // 目标位置
	DestStoreType common.StoreType // 目标存储类型
}
