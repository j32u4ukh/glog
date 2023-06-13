package main

import (
	"fmt"

	"github.com/j32u4ukh/glog/v2"
)

func main() {
	logger0 := glog.SetLogger(0, "a", glog.DebugLevel)
	logger1 := glog.SetLogger(1, "b", glog.DebugLevel)

	fmt.Printf("logger0 idx: %d\n", logger0.GetIdx())
	fmt.Printf("logger1 idx: %d\n", logger1.GetIdx())
	err := glog.UpdateLoggerIndex(0, 2, true)
	if err != nil {
		fmt.Printf("UpdateLoggerIndex err: %s\n", err)
		return
	}
	fmt.Printf("logger0 idx: %d\n", logger0.GetIdx())
	fmt.Printf("logger1 idx: %d\n", logger1.GetIdx())
}
