package controller

import (
	"fmt"
	"log"
)

type Logger interface {
	Debugf(format string, o ...interface{})
	Infof(format string, o ...interface{})
	Warnf(format string, o ...interface{})
	Errorf(err error, format string, o ...interface{})
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
func (df *DefaultLogger) Errorf(err error, format string, o ...interface{}) {
	df.l.Printf("%s: %v", fmt.Sprintf(format, o...), err)
}

var activeLogger Logger = NewDefaultLogger()
