package logger

import "log"

// Printf 输出格式化日志。
func Printf(format string, values ...interface{}) {
	log.Printf(format, values...)
}
