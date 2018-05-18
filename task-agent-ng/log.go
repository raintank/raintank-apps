package main

import (
	"flag"
	"fmt"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	level = flag.Int("log-level", 2, "Set the log level of the application (1=DEBUG|2=INFO|3=WARN|4=ERROR|5=FATAL|6=PANIC)")
)

func InitLogger() {
	switch *level {
	case 1:
		logrus.SetLevel(logrus.DebugLevel)
	case 2:
		logrus.SetLevel(logrus.InfoLevel)
	case 3:
		logrus.SetLevel(logrus.WarnLevel)
	case 4:
		logrus.SetLevel(logrus.ErrorLevel)
	case 5:
		logrus.SetLevel(logrus.FatalLevel)
	case 6:
		logrus.SetLevel(logrus.PanicLevel)
	}
	logrus.AddHook(&locationHook{})
	logrus.Infof("log level set to %v", logrus.GetLevel())
}

type locationHook struct{}

func (*locationHook) Fire(entry *logrus.Entry) error {
	// This is the only way to do this as a hook currently, super messy. Official solution soon hopefully?
	// TODO https://github.com/sirupsen/logrus/issues/63
	_, file, line, ok := runtime.Caller(8)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	entry.Data["source"] = fmt.Sprintf("%s:%d", file, line)

	return nil
}

func (*locationHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}
