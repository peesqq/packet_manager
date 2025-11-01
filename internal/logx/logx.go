package logx

import (
	"fmt"
	"time"
)

func Info(msg string, args ...interface{}) {
	fmt.Printf("%s [INFO] %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(msg, args...))
}

func Warn(msg string, args ...interface{}) {
	fmt.Printf("%s [WARN] %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(msg, args...))
}

func Error(msg string, args ...interface{}) {
	fmt.Printf("%s [ERROR] %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(msg, args...))
}

func Fatal(msg string, args ...interface{}) {
	Error(msg, args...)
	panic(fmt.Sprintf(msg, args...))
}
