package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var sugarLogger *zap.SugaredLogger
var once sync.Once

func InitLogger() {

	// encoder 指定如何写入日志
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   // 以可读的方式展示时间
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // 以大写字母记录日志级别
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	// writeSynce 指定将日志写入文件和控制台
	// file, _ := os.Create("./log/server.log")
	file, _ := os.OpenFile("./log/server.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	writeSyncer := zapcore.AddSync(file)

	fileCore := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel) // 以Debug级别写入
	core := zapcore.NewTee(
		fileCore,
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
	)

	logger := zap.New(core, zap.AddCaller()) // 以功能选项模式注入调用者
	sugarLogger = logger.Sugar()
}

func GetLoggerOr() *zap.SugaredLogger {
	once.Do(InitLogger)
	return sugarLogger
}
