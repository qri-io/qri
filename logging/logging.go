// Package logging is an interface package for logging stuff
// put this file in one place for the entire server package & subpackages
package logging

import (
	"fmt"
)

// Logger interface is to get around the package-level logger anti-pattern
// https://dave.cheney.net/2017/01/23/the-package-level-logger-anti-pattern
// I agree in principle with Dave that importing loggers into packages is an
// anti-pattern, but don't want to pollute custom types with logging details.
// So, we'll still use a global logging variable, and packages need simply
// implement the SetLogger interface with anything that accepts a logger.
// As a default we drop to using fmt-level logging & fmt as a stand-in, defined
// in this file. consumers of this package should call SetLogger
// with a chosen logger
type Logger interface {
	Info(...interface{})
	Infof(string, ...interface{})
	Debug(...interface{})
	Debugf(string, ...interface{})
}

// DefaultLogger is a mega-silly default logger
var DefaultLogger Logger = fmtLogger(0)

// fmtLogger proxies various logging levels as a basic logger
type fmtLogger int

func (fmtLogger) Info(args ...interface{})                  { fmt.Println(append([]interface{}{"INFO"}, args...)...) }
func (fmtLogger) Infof(format string, args ...interface{})  { fmt.Printf("INFO "+format+"\n", args...) }
func (fmtLogger) Debug(args ...interface{})                 {}
func (fmtLogger) Debugf(format string, args ...interface{}) {}
