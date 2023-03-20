package main

import (
	"fmt"
	"runtime"

	"github.com/j32u4ukh/glog"
	"github.com/j32u4ukh/glog/example/internal"
)

func main() {
	logger := glog.GetLogger("../log", "cmd-internal", glog.DebugLevel, false)
	logger.SetOptions(glog.DefaultOption(false, false), glog.UtcOption(8))
	logger.Debug("Start demo2...")
	ptr := logger.CheckCaller(1)
	fmt.Printf("Name: %s\n", runtime.FuncForPC(ptr).Name())
	internal.Init(logger)
	internal.Run()
}
