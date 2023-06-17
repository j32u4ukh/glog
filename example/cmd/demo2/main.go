package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/j32u4ukh/glog/v2"
	"github.com/j32u4ukh/glog/v2/example/internal"
)

func main() {
	logger := glog.SetLogger(0, "cmd-internal", glog.DebugLevel)
	logger.SetFolder("../../log")
	logger.SetOptions(glog.UtcOption(8))
	logger.SetOptions(glog.BasicOption(glog.DebugLevel, true, false, true, true))
	logger.SetOptions(glog.BasicOption(glog.InfoLevel, true, false, true, true))
	logger.SetOptions(glog.BasicOption(glog.WarnLevel, true, true, true, true))
	logger.SetOptions(glog.BasicOption(glog.ErrorLevel, true, true, true, true))
	logger.Debug("Start demo2...")
	err := glog.UpdateLoggerIndex(0, 1, false)
	if err != nil {
		fmt.Printf("UpdateLoggerIndex err: %+v\n", err)
		return
	}
	ptr := logger.CheckCaller(1)
	fmt.Printf("Name: %s\n", runtime.FuncForPC(ptr).Name())
	internal.Init()
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)
		logger.Debug("i: %d", i)
	}
	internal.Run()
}
