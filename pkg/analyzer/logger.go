/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"errors"
	"fmt"
	"log"
)

// Verbosity is an enumerated type for defining the level of verbosity.
type Verbosity int

const (
	LowVerbosity    Verbosity = iota // LowVerbosity only reports errors
	MediumVerbosity                  // MediumVerbosity reports warnings and errors
	HighVerbosity                    // HighVerbosity reports infos, warnings and errors
)

// The Logger interface defines the API for loggers in this package.
type Logger interface {
	Debugf(format string, o ...interface{})
	Infof(format string, o ...interface{})
	Warnf(format string, o ...interface{})
	Errorf(err error, format string, o ...interface{})
}

// DefaultLogger is the package's built-in logger. It uses log.Default() as the underlying logger.
type DefaultLogger struct {
	verbosity Verbosity
	l         *log.Logger
}

// NewDefaultLogger creates an instance of DefaultLogger with the highest verbosity.
func NewDefaultLogger() Logger {
	return NewDefaultLoggerWithVerbosity(HighVerbosity)
}

// NewDefaultLoggerWithVerbosity creates an instance of DefaultLogger with a user-defined verbosity.
func NewDefaultLoggerWithVerbosity(verbosity Verbosity) Logger {
	return &DefaultLogger{
		verbosity: verbosity,
		l:         log.Default(),
	}
}

// Debugf writes a debug message to the log (only if DefaultLogger verbosity is set to HighVerbosity)
func (df *DefaultLogger) Debugf(format string, o ...interface{}) {
	if df.verbosity == HighVerbosity {
		df.l.Printf(format, o...)
	}
}

// Infof writes an informative message to the log (only if DefaultLogger verbosity is set to HighVerbosity)
func (df *DefaultLogger) Infof(format string, o ...interface{}) {
	if df.verbosity == HighVerbosity {
		df.l.Printf(format, o...)
	}
}

// Warnf writes a warning message to the log (unless DefaultLogger verbosity is set to LowVerbosity)
func (df *DefaultLogger) Warnf(format string, o ...interface{}) {
	if df.verbosity >= MediumVerbosity {
		df.l.Printf(format, o...)
	}
}

// Errorf writes an error message to the log (regardless of DefaultLogger's verbosity)
func (df *DefaultLogger) Errorf(err error, format string, o ...interface{}) {
	df.l.Printf("%s: %v", fmt.Sprintf(format, o...), err)
}

func logError(logger Logger, fpe *FileProcessingError) {
	logMsg := fpe.Error().Error()
	location := fpe.Location()
	if location != "" {
		logMsg = fmt.Sprintf("%s, %s", location, logMsg)
	}
	if fpe.IsSevere() || fpe.IsFatal() {
		logger.Errorf(errors.New(logMsg), "")
	} else {
		logger.Warnf(logMsg)
	}
}
