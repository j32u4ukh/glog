package glog

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var loggerMap map[byte]*Logger
var exitChan chan os.Signal

// TODO: v2.0.2 時，取得 Logger ID; 可修改 Logger 在 loggerMap 對應的 idx
// TODO: v2.1.0 時，換檔機制新增: 與開始執行時間點無關，每日零點起算，間隔數小時(0~5: 0; 6~11: 6; 12~17: 12; 18~23: 18)
func init() {
	loggerMap = make(map[byte]*Logger)
	exitChan = make(chan os.Signal)
	signal.Notify(exitChan, os.Interrupt, syscall.SIGTERM)
	go exitHandle()
}

func SetLogger(idx byte, loggerName string, level LogLevel, options ...Option) *Logger {
	var logger *Logger
	var ok bool
	if logger, ok = loggerMap[idx]; ok {
		return logger
	}
	logger = newLogger(loggerName, level, options...)
	loggerMap[idx] = logger
	return logger
}

func GetLogger(idx byte) *Logger {
	if logger, ok := loggerMap[idx]; ok {
		return logger
	}
	return nil
}

func Flush() {
	for _, logger := range loggerMap {
		logger.Flush()
	}
	fmt.Println("glog.Flush | 完成寫出")
}

// 退出時的處理
func exitHandle() {
	<-exitChan
	Flush()
	os.Exit(1)
}
