package log

import (
	"errors"
	"os"
	"tools/loader"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// logger
var infoLogger *zap.Logger
var warnLogger *zap.Logger
var errorLogger *zap.Logger

var level string

const (
	configTableName   = "globalLogConfig"
	infoFile          = "infoFile"
	warnFile          = "warnFile"
	errorFile         = "errorFile"
	messageKey        = "messageKey"
	levelKey          = "levelKey"
	timeKey           = "timeKey"
	nameKey           = "nameKey"
	callerKey         = "callerKey"
	stacktraceKey     = "stacktraceKey"
	maxSize           = "maxSize"
	maxAge            = "maxAge"
	maxBackups        = "maxBackups"
	compress          = "compress"
	logLevel          = "logLevel"
	stdOutput         = "stdOutput"
	openCaller        = "openCaller"
	openFileAndRownum = "openFileAndRownum"
)

// InitLogger 初始化日志
func InitLogger() (success bool, err error) {

	config, err := loader.GetTable(configTableName)

	if err != nil {

		return false, err
	}

	level = loader.StringValue(config, logLevel, "info")

	infoFileName := loader.StringValue(config, infoFile, "")

	// info日志文件不能为空
	if infoFileName == "" {

		return false, errors.New("config information 'infoFile' can not be empty")
	}

	warnFileName := loader.StringValue(config, warnFile, "")

	// warn日志文件不能为空
	if warnFileName == "" {

		return false, errors.New("config information 'warnFile' can not be empty")
	}

	errorFileName := loader.StringValue(config, errorFile, "")

	// warn日志文件不能为空
	if errorFileName == "" {

		return false, errors.New("config information 'errorFile' can not be empty")
	}

	infoHook := lumberjack.Logger{
		Filename:   infoFileName,                              // 文件保存路径
		MaxSize:    loader.IntValue(config, maxSize, 5),       // 每个日志的文件大小, 单位: MB
		MaxAge:     loader.IntValue(config, maxAge, 10),       // 日志文件保存天数
		MaxBackups: loader.IntValue(config, maxBackups, 100),  // 日志文件最多多少个备份
		Compress:   loader.BoolValue(config, compress, false), // 是否压缩
	}

	warnHook := lumberjack.Logger{
		Filename:   warnFileName,
		MaxSize:    loader.IntValue(config, maxSize, 5),
		MaxAge:     loader.IntValue(config, maxAge, 10),
		MaxBackups: loader.IntValue(config, maxBackups, 100),
		Compress:   loader.BoolValue(config, compress, false),
	}

	errorHook := lumberjack.Logger{
		Filename:   errorFileName,
		MaxSize:    loader.IntValue(config, maxSize, 5),
		MaxAge:     loader.IntValue(config, maxAge, 10),
		MaxBackups: loader.IntValue(config, maxBackups, 100),
		Compress:   loader.BoolValue(config, compress, false),
	}

	encoderConfig := zapcore.EncoderConfig{

		MessageKey:     loader.StringValue(config, messageKey, "msg"),
		LevelKey:       loader.StringValue(config, levelKey, "level"),
		TimeKey:        loader.StringValue(config, timeKey, "time"),
		NameKey:        loader.StringValue(config, nameKey, "logger"),
		CallerKey:      loader.StringValue(config, callerKey, "file"),
		StacktraceKey:  loader.StringValue(config, stacktraceKey, "stacktrace"),
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	lv := zapcore.Level(zapcore.DebugLevel)
	levelStr := loader.StringValue(config, logLevel, "info")
	lv.Set(levelStr)

	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(lv)

	infoWrites := []zapcore.WriteSyncer{zapcore.AddSync(&infoHook)}
	warnWrites := []zapcore.WriteSyncer{zapcore.AddSync(&warnHook)}
	errorWrites := []zapcore.WriteSyncer{zapcore.AddSync(&errorHook)}

	isStdOutput := loader.BoolValue(config, stdOutput, false)

	// 在Console上输出
	if isStdOutput {
		infoWrites = append(infoWrites, zapcore.AddSync(os.Stdout))
		warnWrites = append(warnWrites, zapcore.AddSync(os.Stdout))
		errorWrites = append(errorWrites, zapcore.AddSync(os.Stdout))
	}

	encoderType := zapcore.NewJSONEncoder(encoderConfig)

	infoCore := zapcore.NewCore(encoderType, zapcore.NewMultiWriteSyncer(infoWrites...), zap.NewAtomicLevelAt(zapcore.InfoLevel))
	warnCore := zapcore.NewCore(encoderType, zapcore.NewMultiWriteSyncer(warnWrites...), zap.NewAtomicLevelAt(zapcore.WarnLevel))
	errorCore := zapcore.NewCore(encoderType, zapcore.NewMultiWriteSyncer(errorWrites...), zap.NewAtomicLevelAt(zapcore.ErrorLevel))

	// 开启开发模式, 堆栈跟踪
	var caller zap.Option
	openCaller := loader.BoolValue(config, openCaller, false)

	if openCaller {

		caller = zap.AddCaller()
	}

	// 开启文件及行号
	var development zap.Option
	openFileAndRownum := loader.BoolValue(config, openFileAndRownum, false)

	if openFileAndRownum {

		development = zap.Development()
	}

	infoLogger = zap.New(infoCore, caller, development)
	warnLogger = zap.New(warnCore, caller, development)
	errorLogger = zap.New(errorCore, caller, development)

	return true, nil
}

// Info 写info级别日志
func Info(msg string) {

	if level == "info" {

		infoLogger.Info(msg)
	}
}

// Warn 写warn级别日志
func Warn(msg string, writeIntoInfo bool) {

	if level == "info" || level == "warn" {

		if writeIntoInfo {

			infoLogger.Info(msg)
		}

		warnLogger.Warn(msg)
	}
}

// Error 写error级别日志
func Error(msg string, writeIntoInfo bool) {

	if level == "info" || level == "warn" || level == "error" {

		if writeIntoInfo {

			infoLogger.Info(msg)
		}

		errorLogger.Error(msg)
	}
}
