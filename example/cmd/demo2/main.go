package main

import (
	"github.com/j32u4ukh/glog"
	"github.com/j32u4ukh/glog/example/internal"
)

func main() {
	logger := glog.GetLogger("log", "cmd-internal", glog.DebugLevel, false)
	logger.SetOptions(glog.DefaultOption(true, true), glog.UtcOption(8))
	logger.Debug("Start demo2...")
	internal.Init(logger)
	internal.Run()
	logger.Flush()
}
