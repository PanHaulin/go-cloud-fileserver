package handler

import (
	"net/http"

	"gitee.com/porient/go-cloud/v1-gin/code"
	"gitee.com/porient/go-cloud/v1-gin/logger"
	"gitee.com/porient/go-cloud/v1-gin/meta"
	"gitee.com/porient/go-cloud/v1-gin/util"
	"github.com/gin-gonic/gin"
)

const PASSWD_SALT = "123."

// 注册页面接口
func SignupPageHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()
	c.Redirect(http.StatusFound, "/static/view/signup.html")
	sugarLogger.Info("Redirect to signup page")
	// 加载注册页面
	// data, err := ioutil.ReadFile("./static/view/signup.html")
	// if err != nil {
	// 	sugarLogger.Errorf("GET: Failed to load file sign.html, err: %s", err.Error())
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	return
	// }
	// w.Write(data)

}

// POST: 用户注册接口
func SignupHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()
	username := c.Request.FormValue("username")
	passwd := c.Request.FormValue("password")
	// 合法性校验
	if len(username) < 3 || len(passwd) < 5 {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Signup failed",
			"code": code.ParamBindError,
		})
		sugarLogger.Info("Signup Failed, Invalid signup parameters")
		return
	}

	// 加盐加密密码
	encodedPasswd := util.MD5([]byte(passwd + PASSWD_SALT))

	// 保存至数据库
	success := meta.UserSignup(username, encodedPasswd)
	if success {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Signup succeeded",
			"code": 0,
		})
		sugarLogger.Infof("signup succeeded")
	} else {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Signup failed",
			"code": code.MySQLExecError,
		})
		sugarLogger.Errorf("signup Failed, %s", code.Text(code.MySQLExecError))
		return
	}
}

// 用户登录页面接口
func SigninPageHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()
	c.Redirect(http.StatusFound, "/static/view/signin.html")
	sugarLogger.Info("redirect to signin page")
}

// 用户登录接口
func SigninHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()
	username := c.Request.FormValue("username")
	passwd := c.Request.FormValue("password")
	encodedPasswd := util.MD5([]byte(passwd + PASSWD_SALT))
	// 校验用户名和密码
	pwdChecked := meta.UserSignin(username, encodedPasswd)
	if !pwdChecked {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Login failed",
			"code": code.ParamBindError,
		})
		sugarLogger.Info("signup Failed, wrong password")
		return
	}

	// 生成访问凭证 - token
	token, err := meta.GenToken(username)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Login Failed",
			"code": code.ServerError,
		})
		sugarLogger.Errorf("generate token failed, err: %s", err.Error())
		return
	}

	updated := meta.UpdateToken(username, token)
	if !updated {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Login Failed",
			"code": code.MySQLExecError,
		})
		sugarLogger.Errorf("update token failed, err: %s", err.Error())
		return
	}

	// 重定向到主页
	c.JSON(http.StatusOK, gin.H{
		"msg":  "Login succeeded",
		"code": 0,
		"data": struct {
			Location string
			Username string
			Token    string
		}{
			Location: "/static/view/home.html",
			Username: username,
			Token:    token,
		},
	})

	sugarLogger.Info("signin succeeded")
}

func UserInfoHandler(c *gin.Context) {
	sugarLogger := logger.GetLoggerOr()
	// 解析参数
	username := c.Request.FormValue("username")

	// 查询用户信息
	user := meta.GetUserInfo(username)
	user.UserPwd = "" // 将密码置空
	if user == nil {
		// 查询出错
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Failed to get user info",
			"code": code.MySQLExecError,
		})
		sugarLogger.Error("failed to get user info")
		return
	}

	// 组装响应
	c.JSON(http.StatusOK, gin.H{
		"msg":  "OK",
		"code": 0,
		"data": *user,
	})
	sugarLogger.Info("get user info succeeded")
}
