package logger

import (
	"fmt"
	"log"
	"os"
)

var (
	debug *log.Logger
	info  *log.Logger
	err *log.Logger
	warn  *log.Logger
)

func init(){
	debug = log.New(os.Stdout, fmt.Sprintf("%-9s", "[DEBUG]"), log.Ldate|log.Ltime|log.Lshortfile)
	info = log.New(os.Stdout, fmt.Sprintf("%-9s", "[INFO]"), log.Ldate|log.Ltime)
	warn = log.New(os.Stdout, fmt.Sprintf("%-9s", "[WARN]"), log.Ldate|log.Ltime)
	err = log.New(os.Stdout, fmt.Sprintf("%-9s", "[ERROR]"), log.Ldate|log.Ltime)

}

type Logger interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
}

type Log struct {}

func (l *Log)Infof(msg string, args ...interface{}){
	info.Printf(msg, args...)
}

func (l *Log)Debugf(msg string, args ...interface{}){
	debug.Printf(msg, args...)
}

func (l *Log)Warnf(msg string, args ...interface{}){
	warn.Printf(msg, args...)
}

func (l *Log)Errorf(msg string, args ...interface{}){
	err.Printf(msg, args...)
}