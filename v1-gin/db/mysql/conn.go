package mysql

import (
	"time"

	"gitee.com/porient/go-cloud/v1-gin/config"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var db *gorm.DB
var sugarLogger = logger.GetLoggerOr()

func init() {
	dsn := config.MYSQL_SOURCE + "&parseTime=True&loc=Local"

	// 定义mysql, gorm 配置
	db, _ = gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置
	}), &gorm.Config{
		SkipDefaultTransaction: false, // 跳过默认事务，链式操作会被gorm以事务串联起来，保证数据一致性
		NamingStrategy: schema.NamingStrategy{ // 命名策略
			TablePrefix:   "tbl_", // 表名前缀, table for `User` would be `t_users`
			SingularTable: true,   // 使用单数表名, table for `User` would be `user` with this option enabled
			// NoLowerCase:   true,                              // skip the snake_casing of names
			// NameReplacer: strings.NewReplacer("CID", "Cid"), // use name replacer to change struct/field name before convert it to db name
		},
		DisableForeignKeyConstraintWhenMigrating: true, // 避免建立物理外键
	})

	sqlDB, err := db.DB()
	if err != nil {
		sugarLogger.Errorf("Failed to connect to mysql, err: %s", err.Error())
		panic(0)
	}
	// SetMaxIdleConns 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime 设置了连接可复用的最大时间。
	sqlDB.SetConnMaxLifetime(time.Hour)
}

func DBConn() *gorm.DB {
	return db
}
