package sl

import "go.uber.org/zap"

// Err создаёт zap.Field для ошибки
func Err(err error) zap.Field {
	return zap.Error(err)
}

// String создаёт zap.Field для строки
func String(key, val string) zap.Field {
	return zap.String(key, val)
}

// Int создаёт zap.Field для int
func Int(key string, val int) zap.Field {
	return zap.Int(key, val)
}

// Int64 создаёт zap.Field для int64
func Int64(key string, val int64) zap.Field {
	return zap.Int64(key, val)
}

// Any создаёт zap.Field для любого типа
func Any(key string, val any) zap.Field {
	return zap.Any(key, val)
}
