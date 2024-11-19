package log

import (
	"go.uber.org/zap"
)

// levels
const (
	LevelDebug   = zap.DebugLevel
	LevelInfo    = zap.InfoLevel
	LevelWarning = zap.WarnLevel
	LevelError   = zap.ErrorLevel
	LevelStack   = zap.DPanicLevel
	LevelDump    = zap.PanicLevel
	LevelFatal   = zap.FatalLevel
)

func Debug(msg string, args ...zap.Field) {
	zapLogger.Debug(msg, args...)
}

func Info(msg string, args ...zap.Field) {
	zapLogger.Info(msg, args...)
}

func Warning(msg string, args ...zap.Field) {
	zapLogger.Warning(msg, args...)
}

func Error(msg string, args ...zap.Field) {
	zapLogger.Error(msg, args...)
}

func Stack(msg string, args string) {
	zapLogger.Error(msg, zap.Stack(args))
}

func Dump(dump string, args ...zap.Field) {
	zapLogger.Dump(dump, args...)
}

func Fatal(msg string, args ...zap.Field) {
	zapLogger.Fatal(msg, args...)
}

func Close() {
	zapLogger.Close()
}
