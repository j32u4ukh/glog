package main

import (
	"time"

	"github.com/j32u4ukh/glog"
)

func main() {
	logger := glog.GetLogger("log", "not-struct", glog.DebugLevel, false)
	// option1 := glog.BasicOption(glog.DebugLevel, true, true, true)
	// option2 := glog.BasicOption(glog.InfoLevel, true, true, true)
	logger.SetOptions(glog.DefaultOption(false, true))
	logger.SetLogLevel(glog.DebugLevel)

	for t := 0; t < 12; t++ {
		logger.Debug("Hello Debug! t: %d", t)
		logger.Info("Hello Info! t: %d", t)
		logger.Warn("Hello Warn! t: %d", t)
		logger.Error("Hello Error! t: %d", t)
		time.Sleep(time.Second * 1)
	}

	print(logger)

	obj := newObj()
	obj.print("Hello Obj!")
	glog.Flush()
}

func print(logger *glog.Logger) {
	logger.Debug("print Debug!")
	logger.Info("print Info!")
	logger.Warn("print Warn!")
	logger.Error("print Error!")
}

type Obj struct {
	logger *glog.Logger
}

func newObj() *Obj {
	obj := &Obj{
		logger: glog.GetLogger("log", "Obj", glog.DebugLevel, false),
	}
	return obj
}

func (obj *Obj) print(message string) {
	obj.logger.Debug(message)
	obj.logger.Info(message)
	obj.logger.Warn(message)
	obj.logger.Error(message)
}
