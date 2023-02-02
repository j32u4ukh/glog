package glog

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
)

var loggerMap map[string]*Logger
var exitChan chan os.Signal

// TODO: v1.1.1 時，2023/01/29 21:20:28 Info  | [com/j32u4ukh/gos/ans] (*Anser) | 客戶端連接來自: 127.0.0.1:9687 | C:/Users/PC/go/src/gos/ans/anser.go (156)
// 		 其中的 com/j32u4ukh/gos/ans 長度過長
// TODO: v1.2.0 時，換檔機制新增: 與開始執行時間點無關，每日零點起算，間隔數小時(0~5: 0; 6~11: 6; 12~17: 12; 18~23: 18)
// TODO: v2.0.0 時，將建構子中的 callByStruct 移除
// TODO: v2.0.0 時，將建構子中的 options ...Option 移除
func init() {
	loggerMap = make(map[string]*Logger)
	exitChan = make(chan os.Signal)
	signal.Notify(exitChan, os.Interrupt, syscall.SIGTERM)
	go exitHandle()
}

// TODO: 換檔機制新增: 與開始執行時間點無關，每日零點起算，間隔數小時(0~5: 0; 6~11: 6; 12~17: 12; 18~23: 18)
func GetLogger(folder string, loggerName string, level LogLevel, callByStruct bool, options ...Option) *Logger {
	filePath := path.Join(folder, loggerName)
	var logger *Logger
	var ok bool
	if logger, ok = loggerMap[filePath]; ok {
		return logger
	}
	logger = newLogger(folder, loggerName, level, callByStruct, options...)
	loggerMap[filePath] = logger
	return logger
}

func Flush() {
	for _, logger := range loggerMap {
		logger.Flush()
	}
	fmt.Printf("glog.Flush | 完成寫出")
}

// 退出時的處理
func exitHandle() {
	<-exitChan
	Flush()
	os.Exit(1)
}
