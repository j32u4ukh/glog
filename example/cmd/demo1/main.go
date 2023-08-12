package main

import (
	"github.com/j32u4ukh/glog/v2"
)

func main() {
	logger := glog.SetLogger(0, "not-struct", glog.DebugLevel)
	logger.SetFolder("../../log")
	logger.SetOptions(glog.DefaultOption(true, true), glog.UtcOption(8))
	logger.SetLogLevel(glog.DebugLevel)

	for t := 0; t < 50000; t++ {
		logger.Debug("Hello Debug! t: %d", t)
		logger.Info("Hello Info! t: %d", t)
	}

	print(logger)
	glog.Flush()
}

func print(logger *glog.Logger) {
	logger.Debug("print Debug!")
	logger.Info("print Info!")
}
