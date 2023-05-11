package meta

import (
	"time"

	"gitee.com/porient/go-cloud/v1-gin/db/mysql"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"github.com/dgrijalva/jwt-go"
	"gorm.io/gorm"
)

const TOKEN_SALT = "%23&"

type UserToken struct {
	gorm.Model
	Username string `gorm:"type:varchar(64);not null;default:'';comment:'用户名';index:idx_username,unique"`
	Token    string `gorm:"type:text;not null;comment:'用户登录凭证'"`
	User     User   `gorm:"foreignKey:Username;references:Username"`
}

// 需求：需要通过jwt传输的数据
type Claims struct {
	Username string
	jwt.StandardClaims
}

// 生成Token
// func GenToken(username string) string {
// 	// 方案1： (40位=32+8) md5(username+timestamp+token_salt) + timestamp[:8]
// 	ts := fmt.Sprintf("%x", time.Now().Unix())
// 	tokenPrefix := util.MD5([]byte(username + ts + TOKEN_SALT))
// 	token := tokenPrefix + ts[:8]
// 	return token
// }

// 判断token是否有效
// func IsTokenValid(token string) bool {
// 	// 判断token是否过期
// 	// 从db中查询token
// 	// 对比token是否一致
// 	return true
// }

func UpdateToken(username, token string) bool {
	sugarLogger := logger.GetLoggerOr()
	db := mysql.DBConn()
	userToken := UserToken{}

	dbres := db.Where("username = ?", username).Take(&userToken)
	if dbres.Error != nil {
		// 不存在则插入
		if err := db.Create(&UserToken{Username: username, Token: token}).Error; err != nil {
			sugarLogger.Errorf("Failed to create username-token")
			return false
		}
	} else {
		// 存在则更新
		if err := dbres.Update("token", token).Error; err != nil {
			sugarLogger.Errorf("Failed to update token of username = %s, err: %s", username)
			return false
		}
	}
	return true
}

// 生成token :jwt-go
func GenToken(username string) (string, error) {
	// TODO: 以中间件形式改写
	nowTime := time.Now()
	expireTime := nowTime.Add(600 * time.Second)
	claims := Claims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    "go-cloud", // 发行人
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(TOKEN_SALT))
	return token, err
}

// 校验token: jwt-go
func IsTokenValid(token string) bool {
	sugarLogger := logger.GetLoggerOr()
	db := mysql.DBConn()

	// 解析token
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) { return []byte(TOKEN_SALT), nil })
	if err != nil {
		sugarLogger.Errorf("Failed to parse token")
		return false
	}

	// 校验token是否有效
	claims, ok := tokenClaims.Claims.(*Claims)
	if !ok || !tokenClaims.Valid {
		sugarLogger.Errorf("Failed to validate token")
		return false
	}

	// 从数据库中查询token，对比是否一致
	userToken := &UserToken{}
	if err := db.Where("username = ?", claims.Username).Take(userToken).Error; err != nil {
		sugarLogger.Errorf("Parsed token (%s) is not consistent with token (%s) in mysql", token, userToken.Token)
		return false
	}

	sugarLogger.Info("Success to validate token")
	return true
}
