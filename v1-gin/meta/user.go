package meta

import (
	"time"

	"gitee.com/porient/go-cloud/v1-gin/db/mysql"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username       string    `gorm:"type:varchar(64);not null;default:'';comment:'用户名';index:idx_username,unique"`
	UserPwd        string    `gorm:"type:varchar(256);not null;default:'';comment:'用户加密后的密码'"`
	Email          string    `gorm:"type:varchar(64);default:'';comment:'邮箱'"`
	Phone          string    `gorm:"type:varchar(128);default:'';comment:'手机号'"`
	EmailValidated uint      `gorm:"type:tinyint(1);default:0;comment:'邮箱是否验证'"`
	PhoneValidated uint      `gorm:"type:tinyint(1);default:0;comment:'手机号是否验证'"`
	LastActivate   time.Time `gorm:"autoCreateTime;comment:'用户最后活跃时间'"`
	Profile        string    `gorm:"type:text;comment:'用户属性'"`
	Status         int       `gorm:"type:int(11);not null;default:0;comment:'账户状态(启用/禁用/锁定/标记删除等)';index:idx_status"`
}

// 将用户注册信息记录到数据库
func UserSignup(username string, passwd string) bool {
	sugarLogger := logger.GetLoggerOr()
	db := mysql.DBConn()

	dbres := db.Create(&User{Username: username, UserPwd: passwd})
	if dbres.Error != nil {
		// 创建用户失败
		sugarLogger.Errorf("Failed to signup user %s, err: %s", username, dbres.Error)
		return false
	}

	if dbres.RowsAffected <= 0 {
		// 重复插入
		sugarLogger.Warnf("User %s has signup before", username)
		return false
	}
	return true

}

// 获取用户信息并验证密码是否一致
func UserSignin(username, encodedPasswd string) bool {
	sugarLogger := logger.GetLoggerOr()
	db := mysql.DBConn()

	// 查询用户记录
	user := User{}
	dbres := db.Where("username = ?", username).Take(&user)
	if err := dbres.Error; err != nil {
		// ErrRecordNotFound
		sugarLogger.Errorf("Failed to signin: username = %s not found, err: %s", username, err)
		return false
	} else {
		// 更新活跃时间
		dbres.Update("last_activate", time.Now().Format("2006-01-02 15:04:05"))
	}

	if encodedPasswd == user.UserPwd {
		sugarLogger.Infof("username = %s sign in successfully", username)
		return true
	}

	sugarLogger.Infof("Failed to sign in: wrong password for username = %s", username)
	return false
}

func GetUserInfo(username string) *User {
	sugarLogger := logger.GetLoggerOr()
	db := mysql.DBConn()
	user := &User{}
	if err := db.Where("username = ?", username).Take(user).Error; err != nil {
		// ErrRecordNotFound
		sugarLogger.Errorf("Failed to get info: username = %s not found", username)
		return nil
	}
	sugarLogger.Infof("Get info of username = %s", username)
	return user
}
