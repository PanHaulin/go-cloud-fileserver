package meta

import (
	"gitee.com/porient/go-cloud/v1-gin/db/mysql"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"gorm.io/gorm"
)

type UserFile struct {
	gorm.Model
	Username   string   `gorm:"primaryKey;type:varchar(64);not null;index:idx_user_id;index:idx_user_file"`
	FileMD5    string   `gorm:"primaryKey;column:file_md5;type:varchar(64);not null;default:'';comment:'文件hash';index:idx_user_file"`
	FileName   string   `gorm:"type:varchar(256);not null;default:'';comment:'文件别名'"`
	Status     int      `gorm:"tpye:int(11);not null;default:0;comment:'文件状态(0正常1已删除2禁用)';index:idx_status"`
	UniqueFile FileMeta `gorm:"foreignKey:FileMD5;references:FileMD5"` // 关联模式
	// User     User     `gorm:"foreignKey:Username;references:Username"`
}

// 更新用户文件列表
func OnUserFileUploadFinished(username, filehash, filename string) bool {
	db := mysql.DBConn()
	sugarLogger := logger.GetLoggerOr()

	if err := db.Create(&UserFile{Username: username, FileMD5: filehash, FileName: filename}).Error; err != nil {
		sugarLogger.Errorf("Failed to create user-file item, err: %s", err.Error())
		return false
	}
	return true
}

// 批量获取用户文件元信息
func QueryUserFileMetas(username string, limit int) []UserFile {
	sugarLogger := logger.GetLoggerOr()
	db := mysql.DBConn()
	userfiles := make([]UserFile, limit)
	// 需要预加载
	if err := db.Preload("UniqueFile").Limit(limit).Where("username = ?", username).Find(&userfiles).Error; err != nil {
		sugarLogger.Errorf("Failed to get %d files uploaded by %s, err: %s", limit, username, err)
		return nil
	}
	return userfiles
}

// 查询文件被多少用户持有
func NumUsersFileBelongsTo(fileMD5 string) int64 {
	db := mysql.DBConn()
	sugarLogger := logger.GetLoggerOr()

	var count int64
	if err := db.Model(&UserFile{}).Where("file_md5 = ?", fileMD5).Distinct("username").Count(&count).Error; err != nil {
		sugarLogger.Errorf("fail to check number of users the file belongs to, err: %s", err)
		return 0
	}

	return count
}

// 删除特定用户的特定文件的记录
func DelFileOfUser(username, fileMD5 string) {
	db := mysql.DBConn()
	sugarLogger := logger.GetLoggerOr()

	if err := db.Where("username = ? AND file_md5 = ?", username, fileMD5).Delete(&UserFile{}).Error; err != nil {
		sugarLogger.Errorf("fail to delete item: %s", err)
	}
}
