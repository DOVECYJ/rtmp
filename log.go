package rtmp

import "log"

var (
	logger Logger = officialLogger{}
)

func Log(format string, arg ...any) {
	logger.Log(format, arg...)
}

func Warn(format string, arg ...any) {
	logger.Warn(format, arg...)
}

func Error(format string, arg ...any) {
	logger.Error(format, arg...)
}

func Fatal(format string, arg ...any) {
	logger.Fatal(format, arg...)
}

type Logger interface {
	Log(format string, arg ...any)
	Warn(format string, arg ...any)
	Error(format string, arg ...any)
	Fatal(format string, arg ...any)
}

type officialLogger struct {
}

func (l officialLogger) Log(format string, arg ...any) {
	log.Printf(format, arg...)
}

func (l officialLogger) Warn(format string, arg ...any) {
	log.Printf(format, arg...)
}

func (l officialLogger) Error(format string, arg ...any) {
	log.Printf(format, arg...)
}

func (l officialLogger) Fatal(format string, arg ...any) {
	log.Panicf(format, arg...)
}
