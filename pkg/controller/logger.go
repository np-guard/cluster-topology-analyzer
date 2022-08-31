package controller

import (
	"log"
)

type Logger interface {
	Debugf(format string, o ...interface{})
	Infof(format string, o ...interface{})
	Warnf(format string, o ...interface{})
	Errorf(format string, o ...interface{})
}

type DefaultLogger struct {
	l *log.Logger
}

func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		l: log.Default(),
	}
}

func (df *DefaultLogger) Debugf(format string, o ...interface{}) {
	df.l.Printf(format, o...)
}
func (df *DefaultLogger) Infof(format string, o ...interface{}) {
	df.l.Printf(format, o...)
}
func (df *DefaultLogger) Warnf(format string, o ...interface{}) {
	df.l.Printf(format, o...)
}
func (df *DefaultLogger) Errorf(format string, o ...interface{}) {
	df.l.Printf(format, o...)
}

var activeLogger Logger = NewDefaultLogger()
