package glog

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var loggerMap map[byte]*Logger
var exitChan chan os.Signal

// TODO: v2.1.0 時，換檔機制新增: 與開始執行時間點無關，每日零點起算，間隔數小時(0~5: 0; 6~11: 6; 12~17: 12; 18~23: 18)
func init() {
	loggerMap = make(map[byte]*Logger)
	exitChan = make(chan os.Signal)
	signal.Notify(exitChan, os.Interrupt, syscall.SIGTERM)
	go exitHandle()
}

func SetLogger(idx byte, loggerName string, level LogLevel, options ...IOption) *Logger {
	var logger *Logger
	var ok bool
	if logger, ok = loggerMap[idx]; ok {
		return logger
	}
	logger = newLogger(idx, loggerName, level, options...)
	loggerMap[idx] = logger
	return logger
}

func GetLogger(idx byte) *Logger {
	if logger, ok := loggerMap[idx]; ok {
		return logger
	}
	return nil
}

func UpdateLoggerIndex(idx1 byte, idx2 byte, swap bool) error {
	var logger1, logger2 *Logger
	var ok bool
	if logger1, ok = loggerMap[idx1]; !ok {
		return fmt.Errorf("未定義 Logger %d", idx1)
	}
	if swap {
		if logger2, ok = loggerMap[idx2]; !ok {
			return fmt.Errorf("未定義 Logger %d", idx2)
		}
		loggerMap[idx1] = logger2
		logger2.setIdx(idx1)
	} else {
		if logger2, ok = loggerMap[idx2]; ok {
			return fmt.Errorf("已定義 Logger %d", idx2)
		}
		delete(loggerMap, idx1)
	}
	loggerMap[idx2] = logger1
	logger1.setIdx(idx2)
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
