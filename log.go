package faktory_worker

import (
	"log"
	"os"
)

type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, args ...interface{})
	Info(v ...interface{})
	Infof(format string, args ...interface{})
	Warn(v ...interface{})
	Warnf(format string, args ...interface{})
	Error(v ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(v ...interface{})
	Fatalf(format string, args ...interface{})
}

type StdLogger struct {
	*log.Logger
}

func NewStdLogger() Logger {
	flags := log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC
	return &StdLogger{log.New(os.Stdout, "", flags)}
}

func (l *StdLogger) Debug(v ...interface{}) {
	l.Println(v...)
}

func (l *StdLogger) Debugf(format string, v ...interface{}) {
	l.Printf(format+"\n", v...)
}

func (l *StdLogger) Error(v ...interface{}) {
	l.Println(v...)
}

func (l *StdLogger) Errorf(format string, v ...interface{}) {
	l.Printf(format+"\n", v...)
}

func (l *StdLogger) Info(v ...interface{}) {
	l.Println(v...)
}

func (l *StdLogger) Infof(format string, v ...interface{}) {
	l.Printf(format+"\n", v...)
}

func (l *StdLogger) Warn(v ...interface{}) {
	l.Println(v...)
}

func (l *StdLogger) Warnf(format string, v ...interface{}) {
	l.Printf(format+"\n", v...)
}
