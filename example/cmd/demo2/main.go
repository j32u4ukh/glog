package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/j32u4ukh/glog"
	"github.com/j32u4ukh/glog/example/internal"
)

func main() {
	logger := glog.SetLogger(0, "cmd-internal", glog.DebugLevel)
	logger.SetFolder("../../log")
	logger.SetOptions(glog.DefaultOption(false, false), glog.UtcOption(8))
	logger.Debug("Start demo2...")
	ptr := logger.CheckCaller(1)
	fmt.Printf("Name: %s\n", runtime.FuncForPC(ptr).Name())
	internal.Init()
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)
		logger.Debug("i: %d", i)
	}
	internal.Run()
}
