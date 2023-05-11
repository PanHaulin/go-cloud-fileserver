package meta

import (
	"os"
	"sync"

	"gitee.com/porient/go-cloud/v1-gin/db/mysql"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"gitee.com/porient/go-cloud/v1-gin/util"
	"gorm.io/gorm"
)

// 文件元信息
type FileMeta struct {
	gorm.Model
	FileMD5  string `gorm:"column:file_md5;type:char(40);not null;default:'';comment:'文件hash';index:idx_file_hash,unique"`
	FileName string `gorm:"column:file_name;type:varchar(256);not null;default:'';comment:'文件名'"`
	FileSize int64  `gorm:"column:file_size;type:bigint(20);default:0;comment:'文件大小'"`
	Location string `gorm:"column:file_addr;type:varchar(1024);not null;default:'';comment:'文件存储位置'"`
	Status   int    `gorm:"column:status;type:int(11);not null;default:0;comment:'状态(可用/禁用/已删除)';index:idx_status"`
}

// 全局文件信息：索引为MD5
var fileMetas map[string]FileMeta
var mu sync.RWMutex

func init() {
	fileMetas = make(map[string]FileMeta)
}

// 获取文件哈希值
func HashFile(f *os.File) string {
	return util.FileMD5(f)
}

// 设置文件元信息的哈希值
func SetMetaHash(fMeta *FileMeta, fileMd5 string) {
	fMeta.FileMD5 = fileMd5
}

// 新增/更新文件元信息
// func UpdateFileMeta(fMeta FileMeta) {
// 	mu.Lock()
// 	fileMetas[fMeta.FileMD5] = fMeta
// 	mu.Unlock()
// }

// 新增/更新文件元信息到mysql
func UpdateFileMetaDB(fMeta FileMeta) bool {

	sugarLogger := logger.GetLoggerOr()
	db := mysql.DBConn()

	dbres := db.Create(&fMeta) // 插入数据

	if dbres.Error != nil {
		sugarLogger.Errorf("Failed to update file meta, err: %s", &dbres.Error)
		return false
	}

	if dbres.RowsAffected <= 0 {
		// 重复插入
		sugarLogger.Warnf("File with hash: %s has been uploaded before", fMeta.FileMD5)
		return false
	}
	return true
}

// 获取文件元信息
func GetFileMeta(fileMd5 string) FileMeta {
	mu.RLock()
	fMeta := fileMetas[fileMd5]
	mu.RUnlock()
	return fMeta
}

// 从mysql获取文件元信息
func GetFileMetaDB(fileMd5 string) (FileMeta, error) {
	fileMeta := FileMeta{}
	db := mysql.DBConn()
	// 默认使用 Prepared Statement 预编译SQL语句，避免sql注入攻击，提高效率
	dbres := db.Where("file_md5 = ?", fileMd5).Take(&fileMeta)
	return fileMeta, dbres.Error
}

// 删除文件元信息
func RemoveFileMeta(fileMd5 string) {
	mu.Lock()
	delete(fileMetas, fileMd5)
	mu.Unlock()
}
