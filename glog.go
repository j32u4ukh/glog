package glog

import "path"

var loggerMap map[string]*Logger

func init() {
	loggerMap = make(map[string]*Logger)
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
}
