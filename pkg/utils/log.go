package utils

import (
	"log"
	"os"
)

type Logger struct {
	info *log.Logger
}

func NewLogger() *Logger {
	return &Logger{
		info: log.New(os.Stderr, "[IBM CLOUD CNI VPC] - INFO: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (l *Logger) Log(msg string) {
	l.info.Println(msg)
}
