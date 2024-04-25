// Copyright 2024 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"fmt"

	"github.com/juju/loggo/v2"

	"github.com/juju/juju/core/logger"
)

// CheckLogger is an interface that can be used to log messages to a
// *testing.T or *check.C.
type CheckLogger interface {
	Logf(string, ...any)
}

// checkLogger is a loggo.Logger that logs to a *testing.T or *check.C.
type checkLogger struct {
	log  CheckLogger
	name string
}

// WrapCheckLog returns a checkLogger that logs to the given CheckLog.
func WrapCheckLog(log CheckLogger) logger.Logger {
	return checkLogger{log: log}
}

func formatMsg(level, name, msg string) string {
	if name == "" {
		return fmt.Sprintf("%s: ", level) + msg
	}
	return fmt.Sprintf("%s: %s", level, name) + msg
}

func (c checkLogger) Criticalf(msg string, args ...any) {
	c.log.Logf(formatMsg("CRITICAL", c.name, msg), args...)
}

func (c checkLogger) Errorf(msg string, args ...any) {
	c.log.Logf(formatMsg("ERROR", c.name, msg), args...)
}

func (c checkLogger) Warningf(msg string, args ...any) {
	c.log.Logf(formatMsg("WARNING", c.name, msg), args...)
}

func (c checkLogger) Infof(msg string, args ...any) {
	c.log.Logf(formatMsg("INFO", c.name, msg), args...)
}

func (c checkLogger) Debugf(msg string, args ...any) {
	c.log.Logf(formatMsg("DEBUG", c.name, msg), args...)
}

func (c checkLogger) Tracef(msg string, args ...any) {
	c.log.Logf(formatMsg("TRACE", c.name, msg), args...)
}

func (c checkLogger) Logf(level logger.Level, msg string, args ...any) {
	c.log.Logf(formatMsg(loggo.Level(level).String(), c.name, msg), args...)
}

func (c checkLogger) Child(name string) logger.Logger {
	return checkLogger{log: c.log, name: name}
}

func (c checkLogger) ChildWithTags(name string, tags ...string) logger.Logger {
	return checkLogger{log: c.log, name: name}
}

// GetChildByName returns a child logger with the given name.
func (c checkLogger) GetChildByName(name string) logger.Logger {
	return checkLogger{log: c.log, name: name}
}

func (c checkLogger) IsErrorEnabled() bool             { return true }
func (c checkLogger) IsWarningEnabled() bool           { return true }
func (c checkLogger) IsInfoEnabled() bool              { return true }
func (c checkLogger) IsDebugEnabled() bool             { return true }
func (c checkLogger) IsTraceEnabled() bool             { return true }
func (c checkLogger) IsLevelEnabled(logger.Level) bool { return true }
