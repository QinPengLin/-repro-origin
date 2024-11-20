package log

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"time"
)

var (
	OutputType = "all"
	LogPath    = "./log"
	LogSize    = 100
	LogLevel   = LevelDebug
	Encoder    = "console"

	hookDefer      func(entry zapcore.Entry) error
	zapLogger, _   = NewLogger("")
	isSetZapLogger bool
)

// SetLogger It's non-thread-safe
func SetLogger(logger ILogger) {
	if zapLogger != nil && isSetZapLogger == false {
		zapLogger = logger
		isSetZapLogger = true
	}
}

// SetHookDefer 设置日志defer hook 需要在node.Start()之前设置
func SetHookDefer(f func(entry zapcore.Entry) error) {
	if hookDefer == nil {
		hookDefer = f
	}
}

type ILogger interface {
	Debug(msg string, args ...zap.Field)
	Info(msg string, args ...zap.Field)
	Warning(msg string, args ...zap.Field)
	Error(msg string, args ...zap.Field)
	Dump(msg string, args ...zap.Field)
	Fatal(msg string, args ...zap.Field)

	Close()
	SetHook(f func(entry zapcore.Entry) error)
}

type Logger struct {
	logLib *zap.Logger
}

func NewLogger(nodeid string) (ILogger, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}
	//encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder //同步的时候不需要

	//打印格式
	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	if Encoder == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	//日志输出方式
	hookFile := &lumberjack.Logger{
		Filename:  fmt.Sprintf("%s/%s_log.log", LogPath, nodeid), // 日志文件路径
		MaxSize:   LogSize,                                       // 最大日志大小（Mb级别）
		MaxAge:    1,                                             // days
		Compress:  false,                                         // 是否压缩 disabled by default
		LocalTime: true,
	}
	sync := zapcore.AddSync(hookFile)  //设置同步到文件
	console := zapcore.Lock(os.Stderr) //控制台打印
	var cores []zapcore.Core
	switch OutputType { //console|file|all
	case "console":
		cores = append(cores, zapcore.NewCore(encoder, console, LogLevel))
	case "file":
		cores = append(cores, zapcore.NewCore(encoder, sync, LogLevel))
	case "all":
		cores = append(cores, zapcore.NewCore(encoder, console, LogLevel))
		cores = append(cores, zapcore.NewCore(encoder, sync, LogLevel))
	}
	//设置日志对象
	if hookDefer != nil {
		return &Logger{
			logLib: zap.New(
				zapcore.NewTee(cores...),
				zap.AddStacktrace(zap.WarnLevel)).
				WithOptions(zap.AddCaller()).
				With(zap.String("nodeid", nodeid)).
				WithOptions(zap.AddCallerSkip(2),
					zap.Hooks(hookDefer)), //设置日志输出之后
		}, nil
	}
	//无拦截
	return &Logger{
		logLib: zap.New(
			zapcore.NewTee(cores...),
			zap.AddStacktrace(zap.WarnLevel)).
			WithOptions(zap.AddCaller()).
			With(zap.String("nodeid", nodeid)).
			WithOptions(zap.AddCallerSkip(2)),
	}, nil
}

// Close It's dangerous to call the method on logging
func (logger *Logger) Close() {
	logger.logLib.Sync()
}

func (logger *Logger) SetHook(f func(entry zapcore.Entry) error) {
	logger.logLib.WithOptions(zap.Hooks(f))
}

func (logger *Logger) Debug(msg string, args ...zap.Field) {
	logger.logLib.Debug(msg, args...)
}

func (logger *Logger) Info(msg string, args ...zap.Field) {
	logger.logLib.Info(msg, args...)
}

func (logger *Logger) Warning(msg string, args ...zap.Field) {
	logger.logLib.Warn(msg, args...)
}

func (logger *Logger) Error(msg string, args ...zap.Field) {
	logger.logLib.Error(msg, args...)
}

func (logger *Logger) Dump(msg string, args ...zap.Field) {
	logger.logLib.Error(msg, args...)
}

func (logger *Logger) Fatal(msg string, args ...zap.Field) {
	logger.logLib.Fatal(msg, args...)
}
