package log

import (
	"go.uber.org/zap"
	"time"
)

func ErrorAttr(key string, value error) zap.Field {
	return zap.Error(value)
}

func String(key, value string) zap.Field {
	return zap.String(key, value)
}

func Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}

func Int64(key string, value int64) zap.Field {
	return zap.Int64(key, value)
}

func Int32(key string, value int32) zap.Field {
	return zap.Int32(key, value)
}

func Int16(key string, value int16) zap.Field {
	return zap.Int16(key, value)
}

func Int8(key string, value int8) zap.Field {
	return zap.Int8(key, value)
}

func Uint(key string, value uint) zap.Field {
	return zap.Uint(key, value)
}

func Uint64(key string, v uint64) zap.Field {
	return zap.Uint64(key, v)
}

func Uint32(key string, value uint32) zap.Field {
	return zap.Uint32(key, value)
}

func Uint16(key string, value uint16) zap.Field {
	return zap.Uint16(key, value)
}

func Uint8(key string, value uint8) zap.Field {
	return zap.Uint8(key, value)
}

func Float64(key string, v float64) zap.Field {
	return zap.Float64(key, v)
}

func Bool(key string, v bool) zap.Field {
	return zap.Bool(key, v)
}

func Time(key string, v time.Time) zap.Field {
	return zap.Time(key, v)
}

func Duration(key string, v time.Duration) zap.Field {
	return zap.Duration(key, v)
}

func Any(key string, value any) zap.Field {
	return zap.Any(key, value)
}
