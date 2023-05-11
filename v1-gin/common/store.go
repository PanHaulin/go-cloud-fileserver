package common

// 存储类型(表示文件存到哪里)
type StoreType int

const (
	_           StoreType = iota // 0 不使用
	STORE_LOCAL                  // 本地节点
	STORE_CEPH                   // Ceph 集群
	STORE_OSS                    // 阿里OSS
	STORE_MIX                    // 混合(Ceph及OSS)
	STORE_ALL                    // 所有类型的存储都存一份数据
)
